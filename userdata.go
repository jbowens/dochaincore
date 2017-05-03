package dochaincore

import (
	"bytes"
	"text/template"
)

const baseUserData = `
#cloud-config
ssh_authorized_keys:
  - {{.SSHAuthorizedKey}}
users:
  - name: chaincore
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    groups: sudo
    shell: /bin/bash
packages:
  - docker.io
runcmd:
  - mkfs.ext4 -F /dev/disk/by-id/scsi-0DO_Volume_chain-core-storage
  - mkdir -p /mnt/chain-core-storage
  - mount -o discard,defaults /dev/disk/by-id/scsi-0DO_Volume_chain-core-storage /mnt/chain-core-storage
  - echo '/dev/disk/by-id/scsi-0DO_Volume_chain-core-storage /mnt/chain-core-storage ext4 defaults,nofail,discard 0 0' >> /etc/fstab
  - docker run -p 1999:1999 --name dochaincore -v /mnt/chain-core-storage/postgresql/data:/var/lib/postgresql/data -v /mnt/chain-core-storage/logs:/var/log/chain -v /mnt/chain-core-storage/data:/root/.chaincore chaincore/developer
`

type userDataParams struct {
	SSHAuthorizedKey string
}

func buildUserData(keypair *sshKeyPair) (string, error) {
	t, err := template.New("userdata").Parse(baseUserData)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, userDataParams{
		SSHAuthorizedKey: string(keypair.authorizedKey),
	})
	return string(buf.Bytes()), err
}
