# Contract: `ResourceDescriptor.attributes` (proto)

```proto
message ResourceDescriptor {
  // ... fields 1-11 unchanged ...

  // attributes carries the resource's declared properties as a structure,
  // without flattening. It mirrors EstimateCostRequest.attributes.
  // OPTIONAL. Unset or empty means the host sent none; plugins then fall back
  // to tags. tags keep their meaning, and hosts SHOULD keep sending them so
  // plugins that predate this field are unaffected. When a property appears
  // in both with different values, plugins prefer attributes.
  //
  // Host redaction (REQUIRED): hosts MUST omit keys that start with "__",
  // credential-like keys (names containing, case-insensitively, password,
  // secret, token, credential, privatekey, accesskey, or connectionstring),
  // and values the IaC tool marks secret (for Pulumi, the secret signature
  // "4dabf18193072939515e22adb298388d"). Plugins MUST NOT log this field
  // verbatim.
  //
  // Size: the encoded size MUST NOT exceed 65536 bytes (pluginsdk.MaxAttributesBytes).
  // A whole request must also fit the transport limit (1 MB on Connect/HTTP,
  // 4 MB by default on gRPC), so hosts split BatchCost requests whose
  // resources carry large attributes.
  //
  // Numbers are doubles on the wire; send integers above 2^53 as strings.
  // Plugins can read a dotted path with pluginsdk.AttributeValue.
  google.protobuf.Struct attributes = 12;
}
```

Compatibility: an additive field with a new number. `buf breaking` against `main` passes.
