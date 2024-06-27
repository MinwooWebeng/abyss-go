package abyssgo

type AbyssHostCredential struct {
	name             string
	master_key_pkcs8 []byte
}

func NewCredential(name string) (*AbyssHostCredential, error) {
	new_credential := new(AbyssHostCredential)
	new_credential.name = name
	key, err := acrypt.GenerateRSAKeypairPKCS8()
	if err != nil {
		return nil, err
	}
	new_credential.master_key_pkcs8 = key

	return new_credential, nil
}
