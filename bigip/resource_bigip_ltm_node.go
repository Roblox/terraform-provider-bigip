package bigip

import (
	"log"

	"regexp"

	"github.com/f5devcentral/go-bigip"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBigipLtmNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigipLtmNodeCreate,
		Read:   resourceBigipLtmNodeRead,
		//Update: resourceBigipLtmNodeUpdate,
		Delete: resourceBigipLtmNodeDelete,
		Exists: resourceBigipLtmNodeExists,
		Importer: &schema.ResourceImporter{
			State: resourceBigipLtmNodeImporter,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the node",
				ForceNew:     true,
				ValidateFunc: validateF5Name,
			},

			"address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Address of the node",
				ForceNew:    true,
				//ValidateFunc: TODO: validate valid IP address format
			},

			//TODO: more fields!
		},
	}
}

func resourceBigipLtmNodeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*bigip.BigIP)

	name := d.Get("name").(string)
	address := d.Get("address").(string)

	r, _ := regexp.Compile("^((?:[0-9]{1,3}.){3}[0-9]{1,3})|(.*:.*)$")

	log.Println("[INFO] Creating node " + name + "::" + address)
	var err error
	if r.MatchString(address) {
		err = client.CreateNode(
			name,
			address,
		)
	} else {
		err = client.CreateFQDNNode(
			name,
			address,
		)
	}
	if err != nil {
		return err
	}

	d.SetId(name)

	return resourceBigipLtmNodeRead(d, meta)
}

func resourceBigipLtmNodeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*bigip.BigIP)

	name := d.Id()

	log.Println("[INFO] Fetching node " + name)

	node, err := client.GetNode(name)
	if err != nil {
		return err
	}
	if node == nil {
			log.Printf("[WARN] Node (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
	if node.FQDN.Name != "" {
		d.Set("address", node.FQDN.Name)
	} else {
		d.Set("address", node.Address)
	}
	d.Set("name", name)

	return nil
}

func resourceBigipLtmNodeExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*bigip.BigIP)

	name := d.Id()
	log.Println("[INFO] Fetching node " + name)

	node, err := client.GetNode(name)
	if err != nil {
		return false, err
	}

	if node == nil {
		d.SetId("")
	}
	return node != nil, nil
}

func resourceBigipLtmNodeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*bigip.BigIP)

	name := d.Id()

	vs := &bigip.Node{
		Address: d.Get("address").(string),
	}

	return client.ModifyNode(name, vs)
}

func resourceBigipLtmNodeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*bigip.BigIP)

	name := d.Id()
	log.Println("[INFO] Deleting node " + name)

	 err := client.DeleteNode(name)
	if err != nil {
		return err
	}
	if err == nil {
		log.Printf("[WARN] Node (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	regex := regexp.MustCompile("referenced by a member of pool '\\/\\w+/([\\w-_.]+)")
	for err != nil {
		log.Printf("[INFO] Deleting %s from pools...\n", name)
		parts := regex.FindStringSubmatch(err.Error())
		if len(parts) > 1 {
			poolName := parts[1]
			members, e := client.PoolMembers(poolName)
			if e != nil {
				return e
			}
			for _, member := range members.PoolMembers {
				e = client.DeletePoolMember(poolName, member.Name)
				if e != nil {
					return e
				}
			}
			err = client.DeleteNode(name)
		} else {
			break
		}
	}
	return err
}

func resourceBigipLtmNodeImporter(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return []*schema.ResourceData{d}, nil
}
