package tfexporter

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	gcloud "terraform-provider-genesyscloud/genesyscloud"

	resourceExporter "terraform-provider-genesyscloud/genesyscloud/resource_exporter"

	"github.com/hashicorp/go-cty/cty"

	registrar "terraform-provider-genesyscloud/genesyscloud/resource_register"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func SetRegistrar(l registrar.Registrar) {
	l.RegisterResource("genesyscloud_tf_export", ResourceTfExport())

}

func ResourceTfExport() *schema.Resource {
	return &schema.Resource{
		Description: fmt.Sprintf(`
		Genesys Cloud Resource to export Terraform config and (optionally) tfstate files to a local directory. 
		The config file is named '%s' or '%s', and the state file is named '%s'.
		`, defaultTfJSONFile, defaultTfHCLFile, defaultTfStateFile),

		CreateContext: createTfExport,
		ReadContext:   readTfExport,
		DeleteContext: deleteTfExport,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"directory": {
				Description: "Directory where the config and state files will be exported.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "./genesyscloud",
				ForceNew:    true,
			},
			"resource_types": {
				Description: "Resource types to export, e.g. 'genesyscloud_user'. Defaults to all exportable types. NOTE: This field is deprecated and will be removed in future release.  Please use the include_filter_resources or exclude_filter_resources attribute.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: gcloud.ValidateSubStringInSlice(resourceExporter.GetAvailableExporterTypes()),
				},
				ForceNew:      true,
				Deprecated:    "Use include_filter_resources attribute instead",
				ConflictsWith: []string{"include_filter_resources", "exclude_filter_resources"},
			},
			"include_filter_resources": {
				Description: "Include only resources that match either a resource type or a resource type::regular expression.  See export guide for additional information",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: gcloud.ValidateSubStringInSlice(resourceExporter.GetAvailableExporterTypes()),
				},
				ForceNew:      true,
				ConflictsWith: []string{"resource_types", "exclude_filter_resources"},
			},
			"exclude_filter_resources": {
				Description: "Exclude resources that match either a resource type or a resource type::regular expression.  See export guide for additional information",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: gcloud.ValidateSubStringInSlice(resourceExporter.GetAvailableExporterTypes()),
				},
				ForceNew:      true,
				ConflictsWith: []string{"resource_types", "include_filter_resources"},
			},
			"include_state_file": {
				Description: "Export a 'terraform.tfstate' file along with the config file. This can be used for orgs to begin managing existing resources with terraform. When `false`, GUID fields will be omitted from the config file unless a resource reference can be supplied. In this case, the resource type will need to be included in the `resource_types` array.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"export_as_hcl": {
				Description: "Export the config as HCL.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"log_permission_errors": {
				Description: "Log permission/product issues rather than fail.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},
			"exclude_attributes": {
				Description: "Attributes to exclude from the config when exporting resources. Each value should be of the form {resource_name}.{attribute}, e.g. 'genesyscloud_user.skills'. Excluded attributes must be optional.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				ForceNew:    true,
			},
		},
	}
}

type resourceInfo struct {
	State   *terraform.InstanceState
	Name    string
	Type    string
	CtyType cty.Type
}

func createTfExport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if _, ok := d.GetOk("include_filter_resources"); ok {
		gre, _ := NewGenesysCloudResourceExporter(ctx, d, meta, IncludeResources)
		diagErr := gre.Export()
		if diagErr != nil {
			return diagErr
		}

		d.SetId(gre.exportFilePath)
		return nil
	}

	if _, ok := d.GetOk("exclude_filter_resources"); ok {
		gre, _ := NewGenesysCloudResourceExporter(ctx, d, meta, ExcludeResources)
		diagErr := gre.Export()
		if diagErr != nil {
			return diagErr
		}

		d.SetId(gre.exportFilePath)
		return nil
	}

	//Dealing with the traditional resource
	gre, _ := NewGenesysCloudResourceExporter(ctx, d, meta, LegacyInclude)
	diagErr := gre.Export()
	if diagErr != nil {
		return diagErr
	}

	d.SetId(gre.exportFilePath)

	return nil
}

func readTfExport(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// If the output config file doesn't exist, mark the resource for creation.
	path := d.Id()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		d.SetId("")
		return nil
	}
	return nil
}

func deleteTfExport(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	configPath := d.Id()
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("Deleting export config %s", configPath)
		os.Remove(configPath)
	}

	stateFile, _ := getFilePath(d, defaultTfStateFile)
	if _, err := os.Stat(stateFile); err == nil {
		log.Printf("Deleting export state %s", stateFile)
		os.Remove(stateFile)
	}

	tfVarsFile, _ := getFilePath(d, defaultTfVarsFile)
	if _, err := os.Stat(tfVarsFile); err == nil {
		log.Printf("Deleting export vars %s", tfVarsFile)
		os.Remove(tfVarsFile)
	}

	// delete left over folders e.g. prompt audio data
	dir, _ := getFilePath(d, "")
	contents, err := ioutil.ReadDir(dir)
	if err == nil {
		for _, c := range contents {
			if c.IsDir() {
				pathToLeftoverDir := path.Join(dir, c.Name())
				log.Printf("Deleting leftover directory %s", pathToLeftoverDir)
				_ = os.RemoveAll(pathToLeftoverDir)
			}
		}
	}

	return nil
}
