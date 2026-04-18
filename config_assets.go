package updu

import _ "embed"

var (
	// DemoConfigYAML is the canonical full-surface sample configuration shipped with the CLI.
	//go:embed sample.updu.conf
	DemoConfigYAML string

	// TemplateConfigYAML is the canonical starter template with commented examples.
	//go:embed examples/configs/template/updu.conf
	TemplateConfigYAML string
)