package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testHandler struct {
	enabled         bool
	handleCalled    bool
	handleError     error
	lastRecord      *slog.Record
	attrs           []slog.Attr
	group           string
	withAttrCalled  bool
	withGroupCalled bool
}

func newTestHandler(enabled bool, handleError error) *testHandler {
	return &testHandler{
		enabled:     enabled,
		handleError: handleError,
	}
}

func (h *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.enabled
}

func (h *testHandler) Handle(ctx context.Context, record slog.Record) error {
	h.handleCalled = true
	r := record.Clone()
	h.lastRecord = &r
	return h.handleError
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.withAttrCalled = true
	newHandler := newTestHandler(h.enabled, h.handleError)
	newHandler.attrs = append(h.attrs, attrs...)
	return newHandler
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	h.withGroupCalled = true
	newHandler := newTestHandler(h.enabled, h.handleError)
	newHandler.group = name
	return newHandler
}

func TestMultiHandler(t *testing.T) {
	t.Run("NewMultiHandler", func(t *testing.T) {
		h1 := newTestHandler(true, nil)
		h2 := newTestHandler(false, nil)
		h := NewMultiHandler(h1, h2)
		assert.NotNil(t, h)
		assert.NotNil(t, NewMultiHandler())
	})

	t.Run("Enabled", func(t *testing.T) {
		tests := []struct {
			name           string
			handlerStates  []bool
			expectedResult bool
		}{
			{
				name:           "all_disabled",
				handlerStates:  []bool{false, false, false},
				expectedResult: false,
			},
			{
				name:           "one_enabled",
				handlerStates:  []bool{false, true, false},
				expectedResult: true,
			},
			{
				name:           "all_enabled",
				handlerStates:  []bool{true, true, true},
				expectedResult: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var handlers []slog.Handler
				for _, enabled := range tt.handlerStates {
					handlers = append(handlers, newTestHandler(enabled, nil))
				}

				multiHandler := NewMultiHandler(handlers...)
				result := multiHandler.Enabled(context.Background(), slog.LevelInfo)
				assert.Equal(t, tt.expectedResult, result)
			})
		}
	})

	t.Run("Handle", func(t *testing.T) {
		t.Run("different levels", func(t *testing.T) {
			h1 := newTestHandler(true, nil)
			h2 := newTestHandler(false, nil)

			multiHandler := NewMultiHandler(h1, h2)
			ctx := context.Background()
			record := slog.Record{}
			record.AddAttrs(slog.String("key", "value"))

			err := multiHandler.Handle(ctx, record)
			assert.NoError(t, err)

			assert.True(t, h1.handleCalled)
			assert.False(t, h2.handleCalled)

			assert.NotNil(t, h1.lastRecord)
			var gotValue string
			h1.lastRecord.Attrs(func(attr slog.Attr) bool {
				if attr.Key == "key" {
					gotValue = attr.Value.String()
					return false
				}
				return true
			})
			assert.Equal(t, "value", gotValue)
		})

		tests := []struct {
			name          string
			handlerErrors []error
			expectError   bool
		}{
			{
				name:          "no_errors",
				handlerErrors: []error{nil, nil, nil},
				expectError:   false,
			},
			{
				name:          "one_error",
				handlerErrors: []error{nil, errors.New("handler error"), nil},
				expectError:   true,
			},
			{
				name:          "multiple_errors",
				handlerErrors: []error{errors.New("error 1"), errors.New("error 2"), nil},
				expectError:   true,
			},
			{
				name:          "empty_handlers",
				handlerErrors: []error{},
				expectError:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var handlers []slog.Handler
				for _, err := range tt.handlerErrors {
					handlers = append(handlers, newTestHandler(true, err))
				}

				multiHandler := NewMultiHandler(handlers...)
				ctx := context.Background()
				record := slog.Record{}
				record.AddAttrs(slog.String("key", "value"))
				err := multiHandler.Handle(ctx, record)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				for i, h := range handlers {
					testHandler := h.(*testHandler)
					assert.True(t, testHandler.handleCalled, "Handler %d was not called", i)
					if testHandler.lastRecord != nil {
						var gotValue string
						testHandler.lastRecord.Attrs(func(attr slog.Attr) bool {
							if attr.Key == "key" {
								gotValue = attr.Value.String()
								return false
							}
							return true
						})
						assert.Equal(t, "value", gotValue, "Handler %d did not receive correct record", i)
					}
				}
			})
		}
	})

	t.Run("WithAttrs", func(t *testing.T) {
		var buf1, buf2 bytes.Buffer
		h1 := slog.NewTextHandler(&buf1, nil)
		h2 := slog.NewTextHandler(&buf2, nil)
		multiHandler := NewMultiHandler(h1, h2)

		attrs := []slog.Attr{slog.String("service", "api"), slog.Int("port", 8080)}
		newHandler := multiHandler.WithAttrs(attrs)

		logger := slog.New(newHandler)
		logger.Info("server started")

		output1 := buf1.String()
		output2 := buf2.String()

		assert.Contains(t, output1, "msg=\"server started\"")
		assert.Contains(t, output1, "service=api")
		assert.Contains(t, output1, "port=8080")

		assert.Contains(t, output2, "msg=\"server started\"")
		assert.Contains(t, output2, "service=api")
		assert.Contains(t, output2, "port=8080")

		assert.Equal(t, output1, output2)
	})

	t.Run("WithGroup", func(t *testing.T) {
		var buf1, buf2 bytes.Buffer
		h1 := slog.NewTextHandler(&buf1, nil)
		h2 := slog.NewTextHandler(&buf2, nil)
		multiHandler := NewMultiHandler(h1, h2)

		newHandler := multiHandler.WithGroup("request")

		logger := slog.New(newHandler)
		logger.Info("request received", "method", "GET")

		output1 := buf1.String()
		output2 := buf2.String()

		assert.Contains(t, output1, "msg=\"request received\"")
		assert.Contains(t, output1, "request.method=GET")

		assert.Contains(t, output2, "msg=\"request received\"")
		assert.Contains(t, output2, "request.method=GET")

		assert.Equal(t, output1, output2)
	})
}
