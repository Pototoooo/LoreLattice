{{/*
Copyright 2025-2026 Pototoooo
SPDX-License-Identifier: MIT

LoreLattice Helm Chart Template Helpers

Best Practices References:
- https://helm.sh/docs/chart_best_practices/templates/
- https://github.com/argoproj/argo-helm/blob/main/charts/argo-cd/templates/_helpers.tpl
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "lorelattice.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "lorelattice.fullname" -}}
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
Ref: https://helm.sh/docs/chart_best_practices/labels/
*/}}
{{- define "lorelattice.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels following Kubernetes recommended labels.
Ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
*/}}
{{- define "lorelattice.labels" -}}
helm.sh/chart: {{ include "lorelattice.chart" . }}
{{ include "lorelattice.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: lorelattice
{{- end }}

{{/*
Selector labels
*/}}
{{- define "lorelattice.selectorLabels" -}}
app.kubernetes.io/name: {{ include "lorelattice.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component labels - use for individual components
Usage: {{ include "lorelattice.componentLabels" (dict "component" "app" "context" .) }}
*/}}
{{- define "lorelattice.componentLabels" -}}
{{ include "lorelattice.labels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Component selector labels
Usage: {{ include "lorelattice.componentSelectorLabels" (dict "component" "app" "context" .) }}
*/}}
{{- define "lorelattice.componentSelectorLabels" -}}
{{ include "lorelattice.selectorLabels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Create the name of the service account to use.
Ref: https://helm.sh/docs/chart_best_practices/rbac/
*/}}
{{- define "lorelattice.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "lorelattice.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Secret name - supports existing secret
*/}}
{{- define "lorelattice.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "lorelattice.fullname" . }}-secrets
{{- end }}
{{- end }}

{{/*
Return the app image with tag.
Defaults to Chart.appVersion if tag is not specified.
*/}}
{{- define "lorelattice.app.image" -}}
{{- $tag := default .Chart.AppVersion .Values.app.image.tag }}
{{- printf "%s:%s" .Values.app.image.repository $tag }}
{{- end }}

{{/*
Return the frontend image with tag.
*/}}
{{- define "lorelattice.frontend.image" -}}
{{- printf "%s:%s" .Values.frontend.image.repository .Values.frontend.image.tag }}
{{- end }}

{{/*
Return the docreader image with tag.
*/}}
{{- define "lorelattice.docreader.image" -}}
{{- printf "%s:%s" .Values.docreader.image.repository .Values.docreader.image.tag }}
{{- end }}

{{/*
Return the PostgreSQL image with tag.
*/}}
{{- define "lorelattice.postgresql.image" -}}
{{- printf "%s:%s" .Values.postgresql.image.repository .Values.postgresql.image.tag }}
{{- end }}

{{/*
Return the Redis image with tag.
*/}}
{{- define "lorelattice.redis.image" -}}
{{- printf "%s:%s" .Values.redis.image.repository .Values.redis.image.tag }}
{{- end }}

{{/*
Return the Neo4j image with tag.
*/}}
{{- define "lorelattice.neo4j.image" -}}
{{- printf "%s:%s" .Values.neo4j.image.repository .Values.neo4j.image.tag }}
{{- end }}

{{/*
Create image pull secrets list.
*/}}
{{- define "lorelattice.imagePullSecrets" -}}
{{- with .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Return the storage class name.
*/}}
{{- define "lorelattice.storageClass" -}}
{{- if .Values.global.storageClass }}
{{- if eq .Values.global.storageClass "-" }}
storageClassName: ""
{{- else }}
storageClassName: {{ .Values.global.storageClass | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Pod security context.
Merges global defaults with component-specific overrides.
*/}}
{{- define "lorelattice.podSecurityContext" -}}
{{- $global := .Values.global.podSecurityContext | default dict }}
{{- $component := .componentSecurityContext | default dict }}
{{- $merged := merge $component $global }}
{{- if $merged }}
securityContext:
{{- toYaml $merged | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Container security context.
*/}}
{{- define "lorelattice.containerSecurityContext" -}}
{{- if . }}
securityContext:
{{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}
