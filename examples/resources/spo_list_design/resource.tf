# A list design applies one or more site scripts (with list-scoped actions) to a
# list. List designs are immutable — any change recreates the design.
resource "spo_site_script" "contacts_list" {
  title = "Provision contacts list"
  content = jsonencode({
    "$schema" = "schema.json"
    actions = [{
      verb         = "createSPList"
      listName     = "Contacts"
      templateType = 100
      subactions   = [{ verb = "setDescription", description = "Company contacts" }]
    }]
    bindata = {}
    version = 1
  })
}

resource "spo_list_design" "contacts" {
  title        = "Contacts list"
  description  = "Standard contacts list design"
  site_scripts = [spo_site_script.contacts_list.id]
}
