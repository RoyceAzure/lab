package model

type MailContent struct {
	Subject     string
	Content     string
	To          []string
	CC          []string
	BCC         []string
	AttachFiles []string
}
