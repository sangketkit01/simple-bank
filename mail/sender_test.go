package mail

import (
	"log"
	"testing"

	"github.com/sangketkit01/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func TestSendEmailWithGamil(t *testing.T) {
	if testing.Short(){
		t.Skip()
	}
	
	config, err := util.LoadConfig("..")
	require.NoError(t, err)

	sender := NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)

	log.Println(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	subject := "A test email"
	content := `
	<h1>Hello world</h1>
	<p>This is a message from <a href="https://github.com/sangketkit01">Thiaraphat</a></p>
	`

	to := []string{"thiraphat.sa@kkumail.com", "thiraphat_120@hotmail.com"}
	attachFiles := []string{"../test.txt"}

	err = sender.SendEmail(subject, content, to , nil, nil, attachFiles)
	require.NoError(t, err)
}