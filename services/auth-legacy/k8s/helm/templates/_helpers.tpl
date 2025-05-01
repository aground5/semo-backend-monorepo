{{/*
Expand the name of the chart.
*/}}
{{- define "oxhr-authn-server.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "oxhr-authn-server.fullname" -}}
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
{{- define "oxhr-authn-server.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "oxhr-authn-server.labels" -}}
helm.sh/chart: {{ include "oxhr-authn-server.chart" . }}
{{ include "oxhr-authn-server.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
org: growingup
project: oxhr
{{- end }}

{{/*
Selector labels
*/}}
{{- define "oxhr-authn-server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "oxhr-authn-server.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
org: growingup
project: oxhr
{{- end }}