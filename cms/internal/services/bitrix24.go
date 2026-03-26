package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/models"
)

// BitrixClient defines the interface for sending leads to Bitrix24.
type BitrixClient interface {
	SendLead(ctx context.Context, lead models.Lead, webhookURL string) error
}

// HTTPBitrixClient implements BitrixClient using net/http.
type HTTPBitrixClient struct{}

type bitrixPhone struct {
	Value     string `json:"VALUE"`
	ValueType string `json:"VALUE_TYPE"`
}

type bitrixEmail struct {
	Value     string `json:"VALUE"`
	ValueType string `json:"VALUE_TYPE"`
}

type bitrixFields struct {
	Name     string        `json:"NAME"`
	Phone    []bitrixPhone `json:"PHONE"`
	Email    []bitrixEmail `json:"EMAIL,omitempty"`
	Comments string        `json:"COMMENTS"`
	Title    string        `json:"TITLE"`
}

type bitrixPayload struct {
	Fields bitrixFields `json:"fields"`
}

// SendLead sends a lead to Bitrix24 CRM via the provided webhook URL.
func (c *HTTPBitrixClient) SendLead(ctx context.Context, lead models.Lead, webhookURL string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fields := bitrixFields{
		Name:     lead.Name,
		Phone:    []bitrixPhone{{Value: lead.Phone, ValueType: "WORK"}},
		Comments: lead.Comment,
		Title:    "Заявка с сайта " + lead.CreatedAt.Format("02.01.2006 15:04"),
	}

	if lead.Email != "" {
		fields.Email = []bitrixEmail{{Value: lead.Email, ValueType: "WORK"}}
	}

	payload := bitrixPayload{Fields: fields}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal bitrix payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create bitrix request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send bitrix request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("bitrix webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// BitrixPool manages a pool of worker goroutines that process leads asynchronously.
type BitrixPool struct {
	db     *sql.DB
	queue  chan models.Lead
	client BitrixClient
	wg     sync.WaitGroup
}

// NewBitrixPool creates a new BitrixPool with the specified number of workers and queue size.
func NewBitrixPool(db *sql.DB, workers, queueSize int) *BitrixPool {
	p := &BitrixPool{
		db:     db,
		queue:  make(chan models.Lead, queueSize),
		client: &HTTPBitrixClient{},
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

// Submit adds a lead to the processing queue (non-blocking; drops if queue is full).
func (p *BitrixPool) Submit(lead models.Lead) {
	select {
	case p.queue <- lead:
	default:
		// Queue full; drop silently.
	}
}

// Shutdown closes the queue channel and waits for all workers to finish,
// with a maximum wait of the specified timeout.
func (p *BitrixPool) Shutdown(timeout time.Duration) {
	close(p.queue)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// worker processes leads from the queue.
func (p *BitrixPool) worker() {
	defer p.wg.Done()

	for lead := range p.queue {
		p.processLead(lead)
	}
}

// processLead reads current settings and sends the lead to Bitrix24.
func (p *BitrixPool) processLead(lead models.Lead) {
	settings, err := db.Get(p.db)
	if err != nil {
		return
	}

	if !settings.Bitrix24Enabled || settings.Bitrix24Webhook == "" {
		return
	}

	ctx := context.Background()
	sentAt := time.Now().UTC()
	err = p.client.SendLead(ctx, lead, settings.Bitrix24Webhook)

	status := models.StatusSent
	response := ""
	if err != nil {
		status = models.StatusError
		response = err.Error()
		if len(response) > 1000 {
			response = response[:1000]
		}
	}

	_ = db.UpdateLeadBitrix(p.db, lead.ID, status, response, sentAt)
}
