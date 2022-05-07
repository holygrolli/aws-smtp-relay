package relay

import (
	"fmt"
	"net"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

// Client implements the Relay interface.
type Client struct {
	sesAPI          sesiface.SESAPI
	setName         *string
	allowFromRegExp *regexp.Regexp
	denyToRegExp    *regexp.Regexp
}

// Send uses the client SESAPI to send email data
func (c Client) Send(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
) error {
	allowedRecipients, deniedRecipients, err := relay.FilterAddresses(
		from,
		to,
		c.allowFromRegExp,
		c.denyToRegExp,
	)
	if err != nil {
		relay.Log(origin, &from, deniedRecipients, err)
	}
	if len(allowedRecipients) > 0 {
		reFrom := regexp.MustCompile("(?m)[\r\n]+^From:.*$")
		reTo := regexp.MustCompile("(?m)[\r\n]+^To:.*$")
		replaceFrom := fmt.Sprintf("%s%v%s","\r\nFrom: ", *getEnv("FROM", from), "\r\n")
		replaceTo := fmt.Sprintf("%s%v", "\r\nTo: ", *getEnv("TO", fmt.Sprintf("%v",allowedRecipients[0])))
		processedData := []byte(reTo.ReplaceAllString(reFrom.ReplaceAllString(string(data[:]), replaceFrom), replaceTo))
		_, err := c.sesAPI.SendRawEmail(&ses.SendRawEmailInput{
			ConfigurationSetName: c.setName,
			//Source:               getEnv("FROM", from),
			//Destinations:         getEnvArray("TO", allowedRecipients),
			RawMessage:           &ses.RawMessage{Data: processedData},
		})
		relay.Log(origin, getEnv("FROM", from), getEnvArray("TO", allowedRecipients), err)
		//fmt.Println(string(processedData[:]))
		if err != nil {
			return err
		}
	}
	return err
}

// New creates a new client with a session.
func New(
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
) Client {
	return Client{
		sesAPI:          ses.New(session.Must(session.NewSession())),
		setName:         configurationSetName,
		allowFromRegExp: allowFromRegExp,
		denyToRegExp:    denyToRegExp,
	}
}

func getEnv(key, fallback string) *string {
    value, exists := os.LookupEnv(key)
    if !exists {
        value = fallback
    }
    return &value
}

func getEnvArray(key string, fallback []*string) []*string {
    var values []*string
	value, exists := os.LookupEnv(key)
    if !exists {
        values = fallback
    } else {
		values = append(values, &value)
	}
	return values
}