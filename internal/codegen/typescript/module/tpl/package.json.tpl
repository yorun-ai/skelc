{
  "name": "{{ $.PackageName }}",
  "private": true,
  "type": "module",
  "exports": {
    ".": {
      "types": "./index.ts",
      "default": "./index.ts"
    }
  },
  "peerDependencies": {
{{ range $index, $dependency := $.PeerDependencies }}{{ if $index }},
{{ end }}    "{{ $dependency.Package }}": "{{ $dependency.Version }}"{{ end }}
  }
}
