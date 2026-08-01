# skelc Commerce Example

This example complements the minimal quickstart with a multi-domain contract
that exercises the broader Skel language surface.

The `identity.user` domain publishes a user type and permission resource. The
`commerce.order` domain imports that public boundary and demonstrates:

- generic and concrete `data` declarations
- `config`, `event`, `resource`, `service`, `web`, and `task` declarations
- actor authentication and permission support
- local and imported permission requirements
- cross-domain type and resource references
- Go module, TypeScript module, and public Skel generation

From the repository root, run:

```sh
./examples/commerce/generate.sh
```

Generated files are written to `examples/commerce/generated` by default
and are ignored by Git. Pass another directory as the first argument to keep
the output elsewhere:

```sh
./examples/commerce/generate.sh /tmp/skelc-commerce
```

The `identity.user` source contains only public declarations, so it acts as the
producer boundary used by the `commerce.order` import. The script also
generates that boundary as standalone public Skel to demonstrate the artifact
that a separate domain repository would publish.

Generated Go and TypeScript modules declare external Vine and vRPC
dependencies. The script validates generation but does not download or compile
those external dependencies.
