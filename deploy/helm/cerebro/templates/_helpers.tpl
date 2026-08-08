{{/*
Expand the name of the chart.
*/}}
{{- define "cerebro.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified app name (trunc 63, dedupe release name).
*/}}
{{- define "cerebro.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Chart name and version, used by labels.
*/}}
{{- define "cerebro.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels.
*/}}
{{- define "cerebro.labels" -}}
helm.sh/chart: {{ include "cerebro.chart" . }}
{{ include "cerebro.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels.
*/}}
{{- define "cerebro.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cerebro.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Service account name.
*/}}
{{- define "cerebro.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "cerebro.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Image reference (repository:tag, tag falls back to .Chart.AppVersion).
*/}}
{{- define "cerebro.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{/*
Name of the Secret holding the session key and auth passwords: the
chart-managed Secret, unless secret.existingSecret supplies one.
*/}}
{{- define "cerebro.secretName" -}}
{{- .Values.secret.existingSecret | default (include "cerebro.fullname" .) -}}
{{- end -}}

{{/*
Validate values and fail rendering with a clear message on invalid input.
Emits no output on success.
*/}}
{{- define "cerebro.validate" -}}
{{- if not .Values.image.repository -}}
{{- fail "image.repository is required" -}}
{{- end -}}
{{- if gt (int .Values.replicaCount) 1 -}}
{{- fail "replicaCount must be 1: embedded SQLite is a single-writer" -}}
{{- end -}}
{{- if not (or (eq .Values.config.auth.type "") (eq .Values.config.auth.type "basic") (eq .Values.config.auth.type "ldap")) -}}
{{- fail (printf "config.auth.type %q is invalid; must be \"\", \"basic\" or \"ldap\"" .Values.config.auth.type) -}}
{{- end -}}
{{- if and (eq .Values.config.auth.type "basic") (not .Values.secret.existingSecret) }}
  {{- if not .Values.secret.basicAuthPassword }}
    {{- fail "When auth.type=basic, secret.basicAuthPassword is required" }}
  {{- end }}
  {{- if not .Values.config.auth.settings.username }}
    {{- fail "When auth.type=basic, config.auth.settings.username is required" }}
  {{- end }}
{{- end }}
{{- if and (eq .Values.config.auth.type "ldap") (not .Values.secret.existingSecret) }}
  {{- if not .Values.secret.ldapBindPassword }}
    {{- fail "When auth.type=ldap, secret.ldapBindPassword is required" }}
  {{- end }}
{{- end }}
{{- if and (ne .Values.config.auth.type "") (eq .Values.secret.sessionKey "") (eq .Values.secret.existingSecret "") -}}
{{- fail "config.auth.type is set but secret.sessionKey is empty and secret.existingSecret is unset; a session key is required when auth is enabled" -}}
{{- end -}}
{{- end -}}
