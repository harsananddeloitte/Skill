---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "genesyscloud_telephony_providers_edges_phone Data Source - terraform-provider-genesyscloud"
subcategory: ""
description: |-
  Data source for Genesys Cloud Phone. Select a phone by name
---

# genesyscloud_telephony_providers_edges_phone (Data Source)

Data source for Genesys Cloud Phone. Select a phone by name

## Example Usage

```terraform
data "genesyscloud_telephony_providers_edges_phone" "phoneData" {
  name = "test phone name"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **name** (String) Phone name.

### Optional

- **id** (String) The ID of this resource.

