# Contract: Proto

`proto/finfocus/v1/costsource.proto`, `GetActualCostRequest`, appended after field 9:

```protobuf
  // resource describes the resource whose actual cost is requested. It has the
  // same meaning as GetProjectedCostRequest.resource, including
  // ResourceDescriptor.attributes and the host redaction rules on that field.
  //
  // Unset means the host sent no descriptor. Plugins then fall back to tags,
  // resource_id, and arn, as before this field existed.
  //
  // tags keeps its meaning: the resource's cloud tags, usable as billing
  // filters. Hosts SHOULD keep sending them so older plugins are unaffected.
  // When resource is set, plugins read pricing dimensions (provider,
  // resource_type, sku, region, attributes) from it and not from tags; a cloud
  // tag named region, sku, or provider is a label, not a pricing input.
  //
  // resource_id stays the required identifier of this request. Send the same
  // resource on every page of one paginated query. Dry run is otherwise
  // unchanged.
  ResourceDescriptor resource = 11;
```

The field 9 comment's last sentence ("Field 10 is left free for a later billing account name.")
stays, so field 10 is held by comment, not by `reserved`.

Compatibility: additive. `buf breaking` against `main` reports nothing. Old plugins skip field 11 as
an unknown field; old hosts never send it.
