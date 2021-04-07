package logging

import (
	"regexp"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type JSONColorFormatter struct {
	inner *logrus.JSONFormatter
}

func NewJSONColorFormatter(parent string, pretty bool) logrus.Formatter {
	return &JSONColorFormatter{
		inner: &logrus.JSONFormatter{
			DataKey: parent,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime: "timestamp",
				logrus.FieldKeyMsg:  "message",
			},
			PrettyPrint: pretty,
		},
	}
}

var keyRegex = regexp.MustCompile(`"([A-Za-z0-9-_]+)":`)

func (jcf *JSONColorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b, err := jcf.inner.Format(entry)
	if !jcf.inner.PrettyPrint || err != nil {
		return b, err
	}

	result := keyRegex.ReplaceAllFunc(b, func(needle []byte) []byte {
		return []byte(color.HiGreenString("%s", string(needle)))
	})

	return result, err
}
