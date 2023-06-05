{{range .configList}}
- type: filestream
  enabled: true
  paths:
      - {{ .HostDir }}/{{ .File }}
  scan_frequency: 1s
  backoff_factor: 1
  max_backoff: 1s
  fields_under_root: true
  {{if .Stdout}}
  docker-json: false
  {{end}}
  {{if eq .Format "json"}}
  json.keys_under_root: true
  {{end}}
  fields:
      {{range $key, $value := .CustomFields}}
      {{ $key }}: {{ $value }}
      {{end}}
      {{range $key, $value := .Tags}}
      {{ $key }}: {{ $value }}
      {{end}}
      {{range $key, $value := $.container}}
      {{ $key }}: {{ $value }}
      {{end}}
  {{range $key, $value := .CustomConfigs}}
  {{ $key }}: {{ $value }}
  {{end}}
  tail_files: false
  close_inactive: 8h
  close_eof: false
  close_removed: true
  clean_removed: true
  close_renamed: false

{{end}}