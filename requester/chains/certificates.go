package chains

func NewCertificateTestChain(chainContext) (ChainResult, chainContext) {
	cR := ChainResult{ChainName: "new certificate"}

	return cR
}

func RevokeCertificateTestChain(chainContext) (ChainResult, chainContext) {
	cR := ChainResult{ChainName: "revoke certificate"}

	return cR
}