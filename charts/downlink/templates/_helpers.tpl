{{/*
Expand the name of the chart.
*/}}
{{- define "downlink.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
Truncated at 63 chars (DNS naming limit). If release name contains chart name it is used as the full name.
*/}}
{{- define "downlink.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "downlink.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "downlink.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "downlink.selectorLabels" -}}
app.kubernetes.io/name: {{ include "downlink.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Pushgateway URL — bundled service when enabled, otherwise the user-supplied external URL.
*/}}
{{- define "downlink.pushgatewayURL" -}}
{{- if .Values.pushgateway.enabled }}
{{- printf "http://%s-prometheus-pushgateway.%s.svc:9091" .Release.Name .Values.workloadNamespace }}
{{- else }}
{{- .Values.pushgateway.externalUrl }}
{{- end }}
{{- end }}
