package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	diskbufferreader "github.com/trufflesecurity/disk-buffer-reader"

	logContext "github.com/trufflesecurity/trufflehog/v3/pkg/context"
)

func TestArchiveHandler(t *testing.T) {
	tests := map[string]struct {
		archiveURL     string
		expectedChunks int
		matchString    string
	}{
		"gzip-single": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/one-zip.gz",
			1,
			"AKIAYVP4CIPPH5TNP3SW",
		},
		"gzip-nested": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/double-zip.gz",
			1,
			"AKIAYVP4CIPPH5TNP3SW",
		},
		"gzip-too-deep": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/six-zip.gz",
			0,
			"",
		},
		"tar-single": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/one.tar",
			1,
			"AKIAYVP4CIPPH5TNP3SW",
		},
		"tar-nested": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/two.tar",
			1,
			"AKIAYVP4CIPPH5TNP3SW",
		},
		"tar-too-deep": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/six.tar",
			0,
			"",
		},
		"targz-single": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/tar-archive.tar.gz",
			1,
			"AKIAYVP4CIPPH5TNP3SW",
		},
		"gzip-large": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/FifteenMB.gz",
			1543,
			"AKIAYVP4CIPPH5TNP3SW",
		},
		"zip-single": {
			"https://raw.githubusercontent.com/bill-rich/bad-secrets/master/aws-canary-creds.zip",
			1,
			"AKIAYVP4CIPPH5TNP3SW",
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Get(testCase.archiveURL)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			defer resp.Body.Close()

			handler := defaultHandler{}

			newReader, err := diskbufferreader.New(resp.Body)
			if err != nil {
				t.Errorf("error creating reusable reader: %s", err)
			}
			archiveChan, err := handler.HandleFile(logContext.Background(), newReader)
			assert.NoError(t, err)

			count := 0
			re := regexp.MustCompile(testCase.matchString)
			matched := false
			for chunk := range archiveChan {
				count++
				if re.Match(chunk) {
					matched = true
				}
			}

			assert.True(t, matched)
			assert.Equal(t, testCase.expectedChunks, count)
		})
	}
}

func TestOpenInvalidArchive(t *testing.T) {
	reader := strings.NewReader("invalid archive")

	ctx := logContext.AddLogger(context.Background())
	handler := defaultHandler{}

	archiveChan := make(chan []byte)

	err := handler.openArchive(ctx, 0, reader, archiveChan)
	assert.Error(t, err)
}
