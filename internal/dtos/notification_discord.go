package dtos

type NotificationDiscord struct {
	Developer     string `json:"developer"`
	TypeMessage   string `json:"type_message"`
	Process       string `json:"process"`
	File          string `json:"file"`
	NroLine       int    `json:"nro_line"`
	Antecedent    string `json:"antecedent"`
	Exception     string `json:"exception"`
	CustomChannel string `json:"custom_channel"`
}
