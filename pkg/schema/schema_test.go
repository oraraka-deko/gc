package schema_test

import (
	"testing"

	"gc/pkg/schema"
)

func TestRegistry_Lookup(t *testing.T) {
	// 1. Bot API Methods
	methods := []string{"getMe", "sendMessage", "sendPhoto", "answerWebAppQuery", "savePreparedInlineMessage", "savePreparedKeyboardButton"}
	for _, m := range methods {
		s, ok := schema.GlobalRegistry.Get(m)
		if !ok || s.Kind != schema.KindBotAPIMethod {
			t.Errorf("expected Bot API method %q in registry", m)
		}
	}

	// 2. Mini App Components
	components := []string{"BackButton", "BottomButton", "SettingsButton", "HapticFeedback", "CloudStorage", "BiometricManager", "Accelerometer", "DeviceOrientation", "Gyroscope", "LocationManager", "DeviceStorage", "SecureStorage", "Serverless"}
	for _, c := range components {
		s, ok := schema.GlobalRegistry.Get(c)
		if !ok || s.Kind != schema.KindMiniAppComponent {
			t.Errorf("expected Mini App component %q in registry", c)
		}
	}

	// 3. Mini App Types
	types := []string{"WebAppInitData", "ThemeParams", "PopupParams", "SafeAreaInset"}
	for _, typ := range types {
		s, ok := schema.GlobalRegistry.Get(typ)
		if !ok || s.Kind != schema.KindMiniAppType {
			t.Errorf("expected Mini App type %q in registry", typ)
		}
	}

	// 4. Mini App Events
	events := []string{"themeChanged", "viewportChanged", "mainButtonClicked", "invoiceClosed"}
	for _, ev := range events {
		s, ok := schema.GlobalRegistry.Get(ev)
		if !ok || s.Kind != schema.KindMiniAppEvent {
			t.Errorf("expected Mini App event %q in registry", ev)
		}
	}
}

func TestPayloadBuilder_Validation_Success(t *testing.T) {
	sendMsgSchema, ok := schema.GlobalRegistry.Get("sendMessage")
	if !ok {
		t.Fatal("sendMessage schema not found")
	}

	builder := schema.NewPayloadBuilder(sendMsgSchema)
	builder.Set("chat_id", 123456789).
		Set("text", "Hello, Telegram!").
		Set("parse_mode", "HTML")

	payload, err := builder.Build()
	if err != nil {
		t.Fatalf("expected build to succeed, got error: %v", err)
	}

	if payload["chat_id"] != 123456789 || payload["text"] != "Hello, Telegram!" {
		t.Errorf("unexpected payload content: %+v", payload)
	}
}

func TestPayloadBuilder_Validation_MissingRequired(t *testing.T) {
	sendMsgSchema, ok := schema.GlobalRegistry.Get("sendMessage")
	if !ok {
		t.Fatal("sendMessage schema not found")
	}

	builder := schema.NewPayloadBuilder(sendMsgSchema)
	// Missing text
	builder.Set("chat_id", 123456789)

	_, err := builder.Build()
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
}

func TestPayloadBuilder_Validation_AllowedValues(t *testing.T) {
	sendMsgSchema, ok := schema.GlobalRegistry.Get("sendMessage")
	if !ok {
		t.Fatal("sendMessage schema not found")
	}

	builder := schema.NewPayloadBuilder(sendMsgSchema)
	builder.Set("chat_id", 123456789).
		Set("text", "Test").
		Set("parse_mode", "InvalidMode")

	_, err := builder.Build()
	if err == nil {
		t.Fatal("expected error for invalid parse_mode value, got nil")
	}
}

func TestSchema_ToJSONSchema(t *testing.T) {
	sendMsgSchema, ok := schema.GlobalRegistry.Get("sendMessage")
	if !ok {
		t.Fatal("sendMessage schema not found")
	}

	jsonSchema := sendMsgSchema.ToJSONSchema()
	if jsonSchema["title"] != "sendMessage" {
		t.Errorf("unexpected title: %v", jsonSchema["title"])
	}

	props, ok := jsonSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties is not a map: %T", jsonSchema["properties"])
	}

	if _, ok := props["chat_id"]; !ok {
		t.Errorf("chat_id property missing")
	}
	if _, ok := props["text"]; !ok {
		t.Errorf("text property missing")
	}

	jsonStr, err := sendMsgSchema.ToJSONSchemaString()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("failed to generate JSON schema string: %v", err)
	}
}
