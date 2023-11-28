{{/*
Expand the name of the chart.
*/}}
{{- define "k8s-backup-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "k8s-backup-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "k8s-backup-operator.labels" -}}
helm.sh/chart: {{ include "k8s-backup-operator.chart" . }}
{{ include "k8s-backup-operator.selectorLabels" . }}
app.kubernetes.io/created-by: {{ include "k8s-backup-operator.name" . }}
app.kubernetes.io/part-of: {{ include "k8s-backup-operator.name" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "k8s-backup-operator.selectorLabels" -}}
app: ces
k8s.cloudogu.com/part-of: backup
app.kubernetes.io/name: {{ include "k8s-backup-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
