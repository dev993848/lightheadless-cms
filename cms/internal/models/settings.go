package models

// Settings holds the CMS global settings (single row, id=1).
type Settings struct {
	ID               int
	SiteName         string
	AdminEmail       string
	AdminPassword    string
	Bitrix24Webhook  string
	Bitrix24Enabled  bool
}
