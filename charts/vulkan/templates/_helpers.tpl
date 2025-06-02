{{/*
Expand the name of the chart.
*/}}
{{- define "vulkan.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "vulkan.fullname" -}}
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
{{- define "vulkan.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "vulkan.labels" -}}
helm.sh/chart: {{ include "vulkan.chart" . }}
{{ include "vulkan.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "vulkan.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vulkan.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "vulkan.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "vulkan.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/* define ui deployment name */}}
{{- define "vulkan.uiDeploymentName" -}}
{{- printf "%s-ui-deployment" (include "vulkan.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* define ui service name */}}
{{- define "vulkan.uiServiceName" -}}
{{- printf "%s-ui-service" (include "vulkan.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* define api deployment name */}}
{{- define "vulkan.apiDeploymentName" -}}
{{- printf "%s-api" (include "vulkan.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* define api service name */}}
{{- define "vulkan.apiServiceName" -}}
{{- printf "%s-api" (include "vulkan.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* define ui ingress name */}}
{{- define "vulkan.uiIngressName" -}}
{{- printf "%s-ui-ingress" (include "vulkan.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}