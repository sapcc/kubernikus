## Editting Self-Hosted Docs

The self-hosted documentation served by the apiserver is generated using Hugo.

It constists of 2 parts:

  * Hugo Theme in `contrib/kubernikus-docs-builder/data`
  * Markdown docs in `docs`

A live preview for development can be started with:

```
docker run --rm -ti -p 1313:1313 -v $PWD/contrib/kubernikus-docs-builder/data:/data -v $PWD/docs/:/data/content sapcc/kubernikus-docs:latest hugo server --bind 0.0.0.0
```

The docs are then accessible locally on http://localhost:1313
