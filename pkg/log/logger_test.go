package log

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func capturarSaida(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestNewLogger(t *testing.T) {
	tests := map[string]struct {
		options  []LoggerOption
		mensagem string
		nivel    string
		contém   []string
	}{
		"logger_padrao": {
			options:  []LoggerOption{},
			mensagem: "teste de mensagem",
			nivel:    "info",
			contém:   []string{"teste de mensagem", "info"},
		},
		"logger_json": {
			options: []LoggerOption{
				WithFormat("json"),
				WithLevel("debug"),
			},
			mensagem: "mensagem debug",
			nivel:    "debug",
			contém:   []string{"mensagem debug", "debug"},
		},
		"logger_text": {
			options: []LoggerOption{
				WithFormat("text"),
				WithLevel("warn"),
			},
			mensagem: "mensagem warn",
			nivel:    "warn",
			contém:   []string{"mensagem warn", "warn"},
		},
	}

	for nome, tc := range tests {
		t.Run(nome, func(t *testing.T) {
			saida := capturarSaida(func() {
				logger := New(tc.options...)
				switch tc.nivel {
				case "debug":
					logger.Debug(tc.mensagem)
				case "info":
					logger.Info(tc.mensagem)
				case "warn":
					logger.Warn(tc.mensagem)
				case "error":
					logger.Error(tc.mensagem)
				}
			})

			for _, texto := range tc.contém {
				if !strings.Contains(saida, texto) {
					t.Errorf("Saída não contém '%s': %s", texto, saida)
				}
			}
		})
	}
}

func TestLoggerWith(t *testing.T) {
	saida := capturarSaida(func() {
		logger := New(WithFormat("json"))
		contextLogger := logger.With(String("contexto", "valor"))
		contextLogger.Info("mensagem com contexto")
	})

	var log map[string]interface{}
	if err := json.Unmarshal([]byte(saida), &log); err != nil {
		t.Fatalf("Falha ao parsear JSON: %v", err)
	}

	if log["contexto"] != "valor" {
		t.Errorf("Contexto não encontrado no log: %v", log)
	}

	if log["message"] != "mensagem com contexto" {
		t.Errorf("Mensagem incorreta no log: %v", log)
	}
}

func TestLoggerFields(t *testing.T) {
	saida := capturarSaida(func() {
		logger := New(WithFormat("json"))
		logger.Info("mensagem com campos",
			String("string", "valor"),
			Int("inteiro", 123),
			Bool("booleano", true),
			Err(io.EOF),
		)
	})

	var log map[string]interface{}
	if err := json.Unmarshal([]byte(saida), &log); err != nil {
		t.Fatalf("Falha ao parsear JSON: %v", err)
	}

	if log["string"] != "valor" {
		t.Errorf("Campo string não encontrado ou incorreto: %v", log)
	}

	if int(log["inteiro"].(float64)) != 123 {
		t.Errorf("Campo inteiro não encontrado ou incorreto: %v", log)
	}

	if log["booleano"] != true {
		t.Errorf("Campo booleano não encontrado ou incorreto: %v", log)
	}

	if !strings.Contains(log["error"].(string), "EOF") {
		t.Errorf("Campo error não encontrado ou incorreto: %v", log)
	}
}

func TestGlobalLogger(t *testing.T) {
	Global = nil

	saida := capturarSaida(func() {
		Initialize(WithFormat("json"))
		Info("mensagem global")
	})

	if !strings.Contains(saida, "mensagem global") {
		t.Errorf("Mensagem global não encontrada na saída: %s", saida)
	}
}

func TestLogNiveis(t *testing.T) {
	tests := map[string]struct {
		nivel         string
		mensagens     []string
		deveConter    []string
		naoDeveConter []string
	}{
		"nivel_info": {
			nivel:         "info",
			mensagens:     []string{"debug", "info", "warn", "error"},
			deveConter:    []string{"info", "warn", "error"},
			naoDeveConter: []string{"debug"},
		},
		"nivel_warn": {
			nivel:         "warn",
			mensagens:     []string{"debug", "info", "warn", "error"},
			deveConter:    []string{"warn", "error"},
			naoDeveConter: []string{"debug", "info"},
		},
	}

	for nome, tc := range tests {
		t.Run(nome, func(t *testing.T) {
			saida := capturarSaida(func() {
				logger := New(WithLevel(tc.nivel))
				logger.Debug("debug")
				logger.Info("info")
				logger.Warn("warn")
				logger.Error("error")
			})

			for _, texto := range tc.deveConter {
				if !strings.Contains(saida, texto) {
					t.Errorf("Saída deveria conter '%s': %s", texto, saida)
				}
			}

			for _, texto := range tc.naoDeveConter {
				if strings.Contains(saida, texto) {
					t.Errorf("Saída não deveria conter '%s': %s", texto, saida)
				}
			}
		})
	}
}
