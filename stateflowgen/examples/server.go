package examples

//go:generate go run github.com/donutnomad/gogen gen ./...

// @StateFlow(name="Server")
// @Flow: Init           => [ Provisioning ]
// @Flow: Provisioning   => [ Ready(Enabled), Failed ]
// @Flow: Ready(Enabled) => [ (Disabled)! via Updating ]
// @Flow: Ready(Disabled)=> [ (Enabled) ]
// @Flow: Ready(*)       => [ Deleted! via Deleting ]
// @Flow: Failed         => [ Deleted! via Deleting ]
const serverStateFlow = ""
