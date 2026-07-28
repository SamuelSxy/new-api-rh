package system_setting

var TosAccessKey = ""
var TosSecretKey = ""
var TosRegion = "cn-beijing"
var TosBucket = "ai-gc.tos-cn-beijing.volces.com"
var TosEndpoint = "tos-cn-beijing.volces.com"
var TosEnabled = false
// TosPublicRead sets uploaded objects to public-read ACL so upstream model
// services (Ark, Seedance, etc.) can access the files directly via URL.
var TosPublicRead = false
// TosCustomDomain overrides the default bucket URL prefix with a CDN or custom
// domain (e.g. "https://assets.example.com"). Leave empty to use the default
// TOS URL pattern.
var TosCustomDomain = ""
