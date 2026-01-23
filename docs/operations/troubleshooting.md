# Basphere 트러블슈팅

새 환경에 배포할 때 자주 발생하는 문제와 해결 방법입니다.

## 체크리스트

| 단계 | 확인 항목 | 확인 명령어 |
|------|----------|------------|
| 1 | Git 저장소 클론 (sudo 사용) | `ls -la /opt/basphere` |
| 2 | make 설치됨 | `which make` |
| 3 | Go 설치됨 (1.22+) | `go version` |
| 4 | CLI 설치됨 | `which basphere-admin` |
| 5 | API 빌드됨 | `ls /opt/basphere/basphere-api/build/` |
| 6 | config.yaml 설정됨 | `cat /etc/basphere/config.yaml` |
| 7 | vsphere.env 설정됨 (600 권한) | `ls -la /etc/basphere/vsphere.env` |
| 8 | api.yaml bastion 섹션 있음 | `grep -A2 "^bastion:" /etc/basphere/api.yaml` |
| 9 | vSphere 연결 성공 | `curl -k https://vcenter/rest/com/vmware/cis/session` |
| 10 | API 서버 실행 중 | `systemctl status basphere-api` |
| 11 | 웹 폼 접속 가능 | `curl http://localhost:8080/register` |
| 12 | SSH 보안 설정 적용됨 | `sudo sshd -T \| grep passwordauthentication` |
| 13 | fail2ban 실행 중 | `sudo fail2ban-client status sshd` |

---

## 문제 1: 빌드 도구 없음

**증상**: `make: command not found` 또는 `go: command not found`

**원인**: Ubuntu 서버에 make, Go가 기본 설치되어 있지 않음

**해결**:
```bash
sudo apt install -y make golang-go
```

---

## 문제 2: /opt에서 권한 오류

**증상**: `Permission denied` when cloning or building

**원인**: /opt 디렉토리는 root 소유이므로 일반 사용자로 작업 불가

**해결**:
```bash
# Git 클론
cd /opt
sudo git clone https://github.com/your-org/project-basphere.git basphere

# API 빌드 (sudo 필요)
cd /opt/basphere/basphere-api
sudo make tidy && sudo make build-linux
```

---

## 문제 3: 등록 완료 페이지에 "bastion-server" 표시

**증상**: 등록 완료 페이지에 실제 bastion 주소 대신 "bastion-server" 표시

**원인**: api.yaml에 `bastion` 섹션이 누락됨

**해결**:
```bash
# api.yaml에 bastion 섹션 추가
sudo vi /etc/basphere/api.yaml
```

```yaml
bastion:
  address: "bastion.your-company.com"  # 실제 주소로 변경
  port: 50022                           # 실제 포트로 변경
```

```bash
# API 서버 재시작
sudo systemctl restart basphere-api
```

---

## 문제 4: vSphere 인증 실패 (401 Unauthorized)

**증상**: VM 생성 시 "401 Unauthorized" 오류

**원인**: vsphere.env의 인증 정보가 잘못됨

**확인 방법**:
```bash
# vSphere API 직접 테스트
source /etc/basphere/vsphere.env
curl -k -u "${VSPHERE_USER}:${VSPHERE_PASSWORD}" \
  "https://your-vcenter/rest/com/vmware/cis/session" -X POST
```

**해결**:
```bash
# vsphere.env 수정
sudo vi /etc/basphere/vsphere.env

# 인증 정보 확인 후 수정
export VSPHERE_USER='correct-user@domain.local'
export VSPHERE_PASSWORD='correct-password'
```

---

## 문제 5: 사용자 삭제 실패

**증상**: `userdel: user xxx is currently used by process yyy`

**원인**: 해당 사용자의 SSH 세션이 아직 활성화되어 있음

**해결**:
```bash
# 방법 1: 사용자 프로세스 확인 후 대기
ps -u <username>

# 세션이 종료되면 삭제
sudo userdel -r <username>

# 방법 2: 강제 종료 (주의: 사용자 작업 중단됨)
sudo pkill -u <username>
sudo userdel -r <username>
```

---

## 문제 6: SSH 보안 설정이 적용되지 않음

**증상**: `PasswordAuthentication yes`가 여전히 활성화됨

**원인**: sshd_config.d 파일은 알파벳 순으로 로드되어, cloud-init의 `50-cloud-init.conf`가 나중에 로드되면 덮어씌워짐

**확인 방법**:
```bash
# 실제 적용된 설정 확인
sudo sshd -T | grep passwordauthentication

# sshd_config.d 파일 목록 확인
ls -la /etc/ssh/sshd_config.d/
```

**해결**:
```bash
# 파일명을 40-으로 변경하여 50-cloud-init.conf보다 먼저 로드
sudo mv /etc/ssh/sshd_config.d/99-basphere-hardening.conf \
        /etc/ssh/sshd_config.d/40-basphere-hardening.conf

# 또는 cloud-init 설정에서 PasswordAuthentication 제거
sudo vi /etc/ssh/sshd_config.d/50-cloud-init.conf

# SSH 재시작
sudo sshd -t && sudo systemctl restart ssh
```

---

## 문제 7: sudo 비밀번호 요청

**증상**: basphere 사용자로 sudo 실행 시 비밀번호 요청

**해결**:
```bash
# basphere 사용자에게 NOPASSWD 권한 부여
echo "basphere ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/basphere
sudo chmod 440 /etc/sudoers.d/basphere
```

---

## 문제 8: Terraform 폴더 생성 실패

**증상**: VM 생성 시 "folder not found" 오류

**원인**: config.yaml에 지정된 VM 폴더가 vSphere에 존재하지 않음

**해결**:
```bash
# vSphere Web Client에서 폴더 생성
# 또는 govc 사용
govc folder.create /Datacenter/vm/basphere-vms
```

---

## 문제 9: API 서버 연결 실패

**증상**: CLI 명령어 실행 시 "API 서버에 연결할 수 없습니다" 오류

**해결**:
```bash
# API 서버 상태 확인
sudo systemctl status basphere-api

# API 서버 재시작
sudo systemctl restart basphere-api

# 로그 확인
sudo journalctl -u basphere-api -f
```

---

## 문제 10: Terraform 오류

### "network not found"
- `config.yaml`의 `network` 값이 vCenter의 포트그룹 이름과 일치하는지 확인

### "template not found"
- `config.yaml`의 `templates.os.<os>.template` 값이 vCenter의 템플릿 이름과 일치하는지 확인
- 템플릿이 지정된 데이터센터에 있는지 확인

---

## 문제 11: cloud-init 네트워크 설정 문제

### Ubuntu 24.04
- 네트워크 설정은 `guestinfo.metadata` 안에 `network` 키로 포함해야 함
- 별도의 `guestinfo.network`는 작동하지 않음

### Rocky Linux
- 네트워크 인터페이스 이름이 Ubuntu와 다름 (ens33 vs ens192)
- `config.yaml`의 `interface` 설정 확인

---

## 문제 12: 디스크 확장 안 됨

**증상**: VM 디스크가 스펙대로 확장되지 않음

**원인**: Rocky Linux 템플릿에 `cloud-utils-growpart` 미설치

**해결**:
```bash
sudo dnf install -y cloud-utils-growpart
```

---

## 문제 13: yq 설정 읽기 실패

**증상**: `get_config` 함수가 기본값만 반환

**원인**: yq가 snap으로 설치됨 (샌드박스로 /etc 접근 불가)

**해결**:
```bash
# yq가 snap으로 설치되었는지 확인
which yq
# /snap/bin/yq 로 나오면 snap 버전

# snap 버전은 /etc 디렉토리 접근 불가 (샌드박스 제한)
# 해결: snap 제거 후 바이너리로 재설치
sudo snap remove yq
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq
hash -r  # 셸 캐시 초기화
```

---

## 문제 14: IP 할당 실패

**해결**:
```bash
# IP 블록 확인
cat /var/lib/basphere/ipam/allocations.tsv

# IP 사용 현황 확인
cat /var/lib/basphere/ipam/leases.tsv

# 수동 IP 블록 할당
sudo /usr/local/lib/basphere/internal/allocate-block <username>
```

---

## 로그 확인

```bash
# 감사 로그
cat /var/log/basphere/audit.log

# Terraform 로그 (VM별)
cat /var/lib/basphere/terraform/<username>/<vm-name>/terraform-apply.log

# API 서버 로그
sudo journalctl -u basphere-api -f
```

---

## 관련 문서

- [설치 가이드](installation.md)
- [보안 설정](security.md)
