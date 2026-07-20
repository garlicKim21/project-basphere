# OS 이미지 관리 (컨텐츠 라이브러리)

모든 신규 VM은 vCenter 컨텐츠 라이브러리 `os-content-library`의 OVF 아이템에서 직접 배포된다.
인벤토리 VM 템플릿은 더 이상 사용하지 않는다 (기존 VM 하위 호환용 레거시 템플릿 2개만 보존 중).

## 아이템 명명 규칙

`<os>-<버전>-<빌드날짜>` 형식. 같은 OS의 새 빌드는 새 아이템으로 나란히 추가하고,
`/etc/basphere/config.yaml`의 `library_item`만 새 이름으로 바꾸면 신규 VM부터 적용된다.
기존 VM의 main.tf는 생성 시점 아이템을 참조하므로, 해당 VM이 모두 삭제되기 전까지 옛 아이템을 지우지 말 것.

## OS별 이미지 소스

| OS | 소스 | 형식 | 비고 |
|----|------|------|------|
| Ubuntu | cloud-images.ubuntu.com/releases/\<codename\>/release/ | **OVA** | 그대로 라이브러리 업로드. open-vm-tools 내장 |
| Rocky | dl.rockylinux.org GenericCloud-Base | qcow2 | **open-vm-tools 없음** — 주입 필요 (아래) |
| RHEL | access.redhat.com KVM Guest Image | qcow2 | open-vm-tools 없음 + 설치에 subscription 필요 |

⚠️ **핵심**: cloud-init 설정은 guestinfo(open-vm-tools)로 전달되므로, **open-vm-tools가 없는
이미지는 부팅은 되지만 IP/호스트명/SSH키가 설정되지 않는다** (terraform apply가 게스트 IP
대기에서 5분 타임아웃으로 실패). KVM용 qcow2는 모두 여기에 해당한다.

## qcow2 → 라이브러리 아이템 파이프라인

qcow2는 하드웨어 정의가 없어 라이브러리에 바로 배포 가능한 아이템이 될 수 없다.
인벤토리 VM 셸로 하드웨어를 입힌 뒤 라이브러리로 클론하고 셸은 삭제한다.

```bash
# 0. 체크섬 검증 후 open-vm-tools 주입 (Rocky의 경우 - 오프라인 RPM 방식)
#    이 서버의 libguestfs appliance는 네트워크 불통이므로 호스트에서 RPM을 받아 복사
#    필요 패키지: open-vm-tools + libmspack libdrm xmlsec1 xmlsec1-openssl fuse3 fuse3-libs
#                fuse-common dbus-tools libtool-ltdl libxslt pciutils pciutils-libs hwdata
sudo virt-customize -a <image>.qcow2 \
  --copy-in <rpm-dir>:/tmp \
  --run-command 'dnf -y --disablerepo="*" install /tmp/<rpm-dir>/*.rpm; rm -rf /tmp/<rpm-dir>' \
  --run-command 'systemctl enable vmtoolsd.service' \
  --selinux-relabel

# 1. 변환
qemu-img convert -O vmdk -o subformat=streamOptimized <image>.qcow2 <item-name>.vmdk

# 2. 업로드 및 셸 조립 (govc, 인증은 /etc/basphere/vsphere.env)
govc import.vmdk -ds 01-VM-Block <item-name>.vmdk <item-name>/
govc vm.create -ds 01-VM-Block -folder /Basphere/vm/basphere-cli \
  -pool "/Basphere/host/Basphere-Home/Resources" \
  -g <guest_id> -firmware efi -c 2 -m 2048 \
  -net "99-basphere-cli" -net.adapter vmxnet3 \
  -disk "<item-name>/<item-name>.vmdk" -disk.controller pvscsi \
  -on=false <item-name>
govc device.cdrom.add -vm <item-name>
govc vm.markastemplate <item-name>

# 3. 라이브러리로 클론 후 셸 삭제
govc library.clone -ovf -vm <item-name> os-content-library <item-name>
govc vm.destroy <item-name>
```

### RHEL 특수 절차 (subscription 필요)

RHEL 저장소는 등록이 필요한데 libguestfs appliance는 네트워크가 안 되므로,
**vSphere에서 실제 부팅해 설치하는 prep VM 방식**을 쓴다:

1. `virt-customize`로 임시 root SSH 키 + 고정 IP용 oneshot 유닛 주입
   (NM 키파일은 미적용 사례 있음 — `ip` 명령 직접 실행하는 systemd 유닛이 확실)
2. vmdk 변환 → vSphere에서 부팅 → SSH 접속
3. `subscription-manager register --org=<org> --activationkey=<key>` →
   `dnf install open-vm-tools` → `unregister` + `clean`
4. 정리: NM 재활성화, 임시 유닛/키 삭제, root 잠금, `cloud-init clean --logs`,
   `truncate -s 0 /etc/machine-id`, poweroff
5. `markastemplate` → `library.clone` → 셸 삭제

## config.yaml OS 정의 필드

라이브러리 OVF에서는 하드웨어 정보를 읽을 수 없으므로 OS별로 명시한다:

| 필드 | Ubuntu (OVA) | Rocky/RHEL (qcow2 빌드) |
|------|--------------|--------------------------|
| `guest_id` | ubuntu64Guest | rockylinux_64Guest / rhel9_64Guest |
| `firmware` | bios | efi |
| `interface` | **ens192** (OVA는 NIC이 PCI 슬롯 192) | **ens33** (govc vm.create 기본 슬롯 33) |

인터페이스 이름은 NIC의 PCI 슬롯 번호로 결정된다(`ens<슬롯>`). 새 이미지 추가 시
테스트 VM으로 실측 검증할 것 — 틀리면 apply가 게스트 IP 대기 타임아웃으로 실패한다.

## 이미지 스테이징

원본 qcow2와 커스터마이즈본은 `/var/lib/basphere/images/`에 보관.
vmdk는 재생성 가능하므로 공간이 필요하면 삭제해도 된다.
