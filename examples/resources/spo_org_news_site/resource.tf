# The site must be an existing communication site.
resource "spo_org_news_site" "this" {
  identity = "https://contoso.sharepoint.com/sites/news"
}
