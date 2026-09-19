package mail_test

import (
	"context"
	"testing"

	"gin-starter-pack/pkg/mail"

	"github.com/stretchr/testify/assert"
)

func TestNewMailer_DefaultLogMailer(t *testing.T) {
	cfg := &mail.Config{
		Driver:      "log",
		FromAddress: "noreply@example.com",
		FromName:    "Starter Pack",
	}

	mailer := mail.NewMailer(cfg)
	assert.NotNil(t, mailer)

	err := mailer.SendSimple(context.Background(), "user@example.com", "Test Subject", "Test Body", false)
	assert.NoError(t, err)
}

func TestNewMailer_NilConfig(t *testing.T) {
	mailer := mail.NewMailer(nil)
	assert.NotNil(t, mailer)

	msg := &mail.Message{
		To:      []string{"recipient@example.com"},
		Subject: "Welcome",
		Body:    "<h1>Welcome</h1>",
		IsHTML:  true,
	}

	err := mailer.Send(context.Background(), msg)
	assert.NoError(t, err)
}

func TestRenderTemplate_Success(t *testing.T) {
	data := map[string]interface{}{
		"UserName": "John Doe",
		"AppName":  "Starter Pack",
	}

	rendered, err := mail.RenderTemplate("welcome.html", data)
	assert.NoError(t, err)
	assert.Contains(t, rendered, "Welcome to Starter Pack")
	assert.Contains(t, rendered, "John Doe")
}

func TestRenderTemplate_NotFound(t *testing.T) {
	rendered, err := mail.RenderTemplate("non_existent_template.html", nil)
	assert.Error(t, err)
	assert.Empty(t, rendered)
}

func TestSendTemplate_Success(t *testing.T) {
	mailer := mail.NewMailer(nil)
	assert.NotNil(t, mailer)

	data := map[string]interface{}{
		"Name":    "Alice",
		"AppName": "App",
	}

	err := mailer.SendTemplate(context.Background(), "alice@example.com", "Welcome", "welcome.html", data)
	assert.NoError(t, err)
}

