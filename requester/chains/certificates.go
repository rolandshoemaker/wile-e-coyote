package chains

func NewCertificateTestChain(cC ChainContext) (ChainResult, ChainContext) {
	cR := ChainResult{Name: "new certificate"}

	return cR, cC
}

func RevokeCertificateTestChain(cC ChainContext) (ChainResult, ChainContext) {
	cR := ChainResult{Name: "revoke certificate"}

	return cR, cC
}