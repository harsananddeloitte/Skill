---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "genesyscloud_outbound_dnclist Data Source - terraform-provider-genesyscloud"
subcategory: ""
description: |-
  Data source for Genesys Cloud Outbound DNC Lists. Select a DNC list by name.
---

# genesyscloud_outbound_dnclist (Data Source)

Data source for Genesys Cloud Outbound DNC Lists. Select a DNC list by name.

## Example Usage

```terraform
data "genesyscloud_outbound_dnclist" "dnc_list" {
  name = "Example DNC List"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) DNC List name.

### Read-Only

- `id` (String) The ID of this resource.

