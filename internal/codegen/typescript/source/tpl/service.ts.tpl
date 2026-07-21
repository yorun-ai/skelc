{{- if $.Services }}
import type {
  VrpcClient,
  VrpcRequestOptions,
} from '@yorun-ai/vrpc';
import {
{{- range $service := $.Services }}
  {{ $service.SpecName }},
{{- end }}
} from './spec';
{{- if $.TypeImports }}
import type {
{{- range $typeName := $.TypeImports }}
  {{ $typeName }},
{{- end }}
} from './data';
{{- end }}
{{- range $import := $.ExternalTypeImports }}
import type * as {{ $import.Alias }} from '{{ $import.Path }}';
{{- end }}

{{- range $serviceIndex, $service := $.Services }}
{{- if gt $serviceIndex 0 }}

{{- end }}
{{- if $service.CommentLines }}
/**
{{- range $line := $service.CommentLines }}
 * {{ $line }}
{{- end }}
 */
{{- end }}
export function {{ $service.FactoryName }}(client: VrpcClient) {
  return {
  {{- range $methodIndex, $method := $service.Methods }}
    {{- if gt $methodIndex 0 }}

    {{- end }}
    {{- if or $method.SummaryLines $method.ParamDocs $method.ReturnDoc }}
    /**
    {{- range $line := $method.SummaryLines }}
     * {{ $line }}
    {{- end }}
    {{- range $paramDoc := $method.ParamDocs }}
     * @param {{ $paramDoc.Name }} - {{ $paramDoc.Description }}
    {{- end }}
    {{- if $method.ReturnDoc }}
     * @returns {{ $method.ReturnDoc.TypeName }} - {{ $method.ReturnDoc.Description }}
    {{- end }}
     */
    {{- end }}
    {{ $method.Name }}(
      params: {{- if $method.HasParams }} {
      {{- range $argument := $method.Arguments }}
        {{ $argument.Name }}: {{ $argument.Type.Plain }};
      {{- end }}
      }{{- else }} null{{- end }},
      options?: VrpcRequestOptions,
    ) {
      return client.invoke<{{ $method.ReturnType }}>({
        serviceName: {{ $service.SpecName }}.serviceName,
        methodName: {{ $service.SpecName }}.methods.{{ $method.Name }},
        params,
        {{- if $method.HasWire }}
        options: {
          ...options,
          wire: {{ $service.SpecName }}.wire.{{ $method.Name }},
        },
        {{- else }}
        options,
        {{- end }}
      });
    },
  {{- end }}
  };
}
{{- end }}
{{- else }}
export {};
{{- end }}
