package settings

func defaults() Settings {
	return Settings{
		ConfigPath:  DefaultConfigPath,
		SessionsDir: DefaultSessionsDir,
		Providers: []ProviderSettings{
			{Name: ProviderAnthropic, API: ProviderAnthropic, APIKeyEnv: "ANTHROPIC_API_KEY"},
			{Name: ProviderOpenAI, API: ProviderOpenAI, APIKeyEnv: "OPENAI_API_KEY"},
			{Name: ProviderGoogle, API: ProviderGoogle, APIKeyEnv: "GOOGLE_API_KEY"},
		},
	}
}
