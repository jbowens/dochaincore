package dochaincore

const baseUserData = `
#cloud-config
package_upgrade: true
users:
  - name: chaincore
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    groups: sudo
    shell: /bin/bash
packages:
  - htop
  - tree
  - docker.io
runcmd:
  - mkfs.ext4 -F /dev/disk/by-id/scsi-0DO_Volume_chain-core-storage
  - mkdir -p /mnt/chain-core-storage
  - mount -o discard,defaults /dev/disk/by-id/scsi-0DO_Volume_chain-core-storage /mnt/chain-core-storage
  - echo '/dev/disk/by-id/scsi-0DO_Volume_chain-core-storage /mnt/chain-core-storage ext4 defaults,nofail,discard 0 0' >> /etc/fstab
  - docker run -p 1999:1999 -v /mnt/chain-core-storage/postgresql/data:/var/lib/postgresql/data chaincore/developer
`

func buildUserData(opt *options) (string, error) {
	return baseUserData, nil
}
