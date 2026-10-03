import { create, clone, type JsonObject } from "@bufbuild/protobuf";
import { ResourceDescriptor, ResourceDescriptorSchema } from "../generated/finfocus/v1/costsource_pb.js";
import { ValidationError } from "../errors/validation-error.js";

export class ResourceDescriptorBuilder {
  private descriptor: ResourceDescriptor;

  constructor() {
    this.descriptor = create(ResourceDescriptorSchema);
  }

  withProvider(provider: string): this {
    if (!provider || provider.trim() === '') {
      throw new ValidationError("Provider cannot be empty", "provider");
    }
    this.descriptor.provider = provider;
    return this;
  }

  withResourceType(type: string): this {
    if (!type || type.trim() === '') {
      throw new ValidationError("Resource type cannot be empty", "resourceType");
    }
    this.descriptor.resourceType = type;
    return this;
  }

  withRegion(region: string): this {
    if (!region || region.trim() === '') {
      throw new ValidationError("Region cannot be empty", "region");
    }
    this.descriptor.region = region;
    return this;
  }

  withSku(sku: string): this {
    if (!sku || sku.trim() === '') {
      throw new ValidationError("SKU cannot be empty", "sku");
    }
    this.descriptor.sku = sku;
    return this;
  }

  withTags(tags: { [key: string]: string }): this {
    this.descriptor.tags = tags;
    return this;
  }

  /**
   * Sets the resource's declared properties as a nested object, without
   * flattening. Hosts must redact secrets and credential-like keys first, and
   * the encoded size must not exceed 65536 bytes. Plugins fall back to tags
   * when attributes are absent.
   */
  withAttributes(attributes: JsonObject): this {
    this.descriptor.attributes = structuredClone(attributes);
    return this;
  }

  withArn(arn: string): this {
    if (!arn || arn.trim() === '') {
      throw new ValidationError("ARN cannot be empty", "arn");
    }
    this.descriptor.arn = arn;
    return this;
  }

  withId(id: string): this {
    if (!id || id.trim() === '') {
      throw new ValidationError("ID cannot be empty", "id");
    }
    this.descriptor.id = id;
    return this;
  }

  build(): ResourceDescriptor {
    // Create a copy to prevent mutation issues when reusing the builder
    return clone(ResourceDescriptorSchema, this.descriptor);
  }
}
