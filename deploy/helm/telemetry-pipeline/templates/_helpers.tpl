{{/*
Expand the name of the chart.
*/}}
{{- define "telemetry-pipeline.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "telemetry-pipeline.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "telemetry-pipeline.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "telemetry-pipeline.labels" -}}
helm.sh/chart: {{ include "telemetry-pipeline.chart" . }}
{{ include "telemetry-pipeline.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "telemetry-pipeline.selectorLabels" -}}
app.kubernetes.io/name: {{ include "telemetry-pipeline.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Telemetry Streamer labels
*/}}
{{- define "telemetry-pipeline.streamer.labels" -}}
{{ include "telemetry-pipeline.labels" . }}
app.kubernetes.io/component: streamer
{{- end }}

{{/*
Telemetry Streamer selector labels
*/}}
{{- define "telemetry-pipeline.streamer.selectorLabels" -}}
{{ include "telemetry-pipeline.selectorLabels" . }}
app.kubernetes.io/component: streamer
{{- end }}

{{/*
Telemetry Collector labels
*/}}
{{- define "telemetry-pipeline.collector.labels" -}}
{{ include "telemetry-pipeline.labels" . }}
app.kubernetes.io/component: collector
{{- end }}

{{/*
Telemetry Collector selector labels
*/}}
{{- define "telemetry-pipeline.collector.selectorLabels" -}}
{{ include "telemetry-pipeline.selectorLabels" . }}
app.kubernetes.io/component: collector
{{- end }}

{{/*
API Gateway labels
*/}}
{{- define "telemetry-pipeline.api-gateway.labels" -}}
{{ include "telemetry-pipeline.labels" . }}
app.kubernetes.io/component: api-gateway
{{- end }}

{{/*
API Gateway selector labels
*/}}
{{- define "telemetry-pipeline.api-gateway.selectorLabels" -}}
{{ include "telemetry-pipeline.selectorLabels" . }}
app.kubernetes.io/component: api-gateway
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "telemetry-pipeline.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "telemetry-pipeline.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create container image reference
*/}}
{{- define "telemetry-pipeline.image" -}}
{{- $registry := .registry | default "docker.io" -}}
{{- $repository := .repository -}}
{{- $tag := .tag | default "latest" | toString -}}
{{- printf "%s/%s:%s" $registry $repository $tag -}}
{{- end }}