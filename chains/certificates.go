// ┬ ┬┬┬  ┌─┐  ┌─┐  ┌─┐┌─┐┬ ┬┌─┐┌┬┐┌─┐
// │││││  ├┤───├┤───│  │ │└┬┘│ │ │ ├┤ 
// └┴┘┴┴─┘└─┘  └─┘  └─┘└─┘ ┴ └─┘ ┴ └─┘
//

package chains

func NewCertificateTestChain() (ChainResult) {
	// check that there are authorizations to do this with or return empty CR so attacker can continue
	var cR ChainResult

	cR.Name = "new certificate"

	return cR
}

func RevokeCertificateTestChain() (ChainResult) {
	// check that there are certificates to do this with or return empty CR so attacker can continue
	var cR ChainResult

	cR.Name = "revoke certificate"

	return cR
}