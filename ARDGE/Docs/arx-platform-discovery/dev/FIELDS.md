# TXT 記錄、Device Info、DiscoveryHandler 欄位映射

## 1️⃣ TXT 記錄欄位 (mDNS 廣告層)
### 來源: device/info.go ToTXTRecords()

| TXT Key | TXT Value 範例 | 資料型別 |
|---------|----------------|--------|
| `device_id` | `device-9e:81:af:e0:5c:84` | string |
| `hostname` | `docker-desktop.local.` | string |
| `model` | `linux-arm64` | string |
| `bios_ver` | `unknown` | string |
| `platform_ver` | `24.04` | string |
| `cpu_cores` | `8` | string (needs parsing) |
| `memory` | `12529836032` | string (needs parsing) |
| `api_port` | `8080` | string (needs parsing) |
| `tls` | `true` | string (needs parsing) |
| `if0_name` | `eth0` | string |
| `if0_ipv4` | `192.168.65.3` | string |
| `if0_mac` | `9e:81:af:e0:5c:84` | string |
| `if0_role` | `PRIMARY`/`SECONDARY` | string |
| `if1_name` | `services1` | string |
| `if1_ipv4` | `192.168.65.6` | string |
| `if1_mac` | `26:37:9a:d5:52:10` | string |
| `if1_role` | `SECONDARY` | string |
| ... | ... (支援 if0 ~ if9) | ... |

---

## 2️⃣ Device Info 資料欄位 (領域模型層)
### 來源: internal/device/types.go DeviceInfo 結構

| 欄位名稱 | 型別 | 對應 TXT Key |
|---------|------|-------------|
| `DeviceID` | string | `device_id` |
| `Hostname` | string | `hostname` |
| `ModelName` | string | `model` |
| `BIOSVersion` | string | `bios_ver` |
| `PlatformVersion` | string | `platform_ver` |
| `CPUCores` | int | `cpu_cores` |
| `MemoryBytes` | int64 | `memory` |
| `APIPort` | int | `api_port` |
| `TLSSupport` | bool | `tls` |
| `Interfaces` | []NetworkInterface | `if{n}_*` |

### NetworkInterface 子結構

| 欄位名稱 | 型別 | 對應 TXT Key |
|---------|------|-------------|
| `Name` | string | `if{n}_name` |
| `IPv4` | string | `if{n}_ipv4` |
| `MAC` | string | `if{n}_mac` |
| `Role` | InterfaceRole | `if{n}_role` |

---

## 3️⃣ DiscoveryHandler EdgeDevice 欄位 (gRPC Protobuf 層)
### 來源: internal/server/discovery.go serviceEntryToEdgeDevice()

| Protobuf 欄位 | 型別 | 對應 Device Info | 對應 TXT Key | 轉換邏輯 |
|--------------|------|-----------------|------------|---------|
| `device_id` | string | DeviceID | `device_id` | 直接賦值 |
| `hostname` | string | Hostname | `hostname` | 直接賦值 |
| `model_name` | string | ModelName | `model` | 直接賦值 |
| `bios_version` | string | BIOSVersion | `bios_ver` | 直接賦值 |
| `platform_version` | string | PlatformVersion | `platform_ver` | 直接賦值 |
| `api_port` | int32 | APIPort | `api_port` | strconv.ParseInt() |
| `tls_support` | bool | TLSSupport | `tls` | `== "true"` 比較 |
| `cpu_cores` | int32 | CPUCores | `cpu_cores` | strconv.ParseInt() |
| `memory` | int64 | MemoryBytes | `memory` | strconv.ParseInt() |
| `interfaces` | []NetworkInterface | Interfaces | `if{n}_*` | buildNetworkInterfaces() |

### NetworkInterface 在 EdgeDevice 中

| Protobuf 欄位 | 型別 | 對應 Device Info | 轉換邏輯 |
|--------------|------|-----------------|---------|
| `name` | string | Name | 直接賦值 |
| `ipv4` | string | IPv4 | 直接賦值 |
| `mac` | string | MAC | 直接賦值 |
| `role` | NetworkInterfaceRole enum | Role | parseInterfaceRole() 轉換 |

---

## 完整轉換流程示例

```
TXT 記錄 (廣告層)
================
device_id=device-xyz
model=linux-arm64
cpu_cores=8
memory=12529836032
api_port=8080
tls=true
if0_name=eth0
if0_ipv4=192.168.65.3
if0_mac=9e:81:af:e0:5c:84
if0_role=SECONDARY

        ↓ Scanner.parseEntry() 解析
        
Device Info (領域模型層)
======================
DeviceID:        "device-xyz"
ModelName:       "linux-arm64"
CPUCores:        8
MemoryBytes:     12529836032
APIPort:         8080
TLSSupport:      true
Interfaces: [
  {
    Name:  "eth0",
    IPv4:  "192.168.65.3",
    MAC:   "9e:81:af:e0:5c:84",
    Role:  RoleSecondary,
  }
]

        ↓ DiscoveryHandler.serviceEntryToEdgeDevice() 轉換
        
Protobuf EdgeDevice (gRPC 層)
=============================
{
  "deviceId": "device-xyz",
  "modelName": "linux-arm64",
  "cpuCores": 8,
  "memory": "12529836032",
  "apiPort": 8080,
  "tlsSupport": true,
  "interfaces": [
    {
      "name": "eth0",
      "ipv4": "192.168.65.3",
      "mac": "9e:81:af:e0:5c:84",
      "role": "NETWORK_INTERFACE_ROLE_SECONDARY"
    }
  ]
}

        ↓ JSON 序列化返回客戶端
```

---

## 轉換函式位置

| 轉換階段 | 函式 | 檔案 | 行號 |
|---------|------|------|------|
| TXT → ServiceEntry | `Scanner.parseEntry()` | scanner.go | 237 |
| ServiceEntry → EdgeDevice | `DiscoveryHandler.serviceEntryToEdgeDevice()` | discovery.go | 79 |
| ServiceEntry → Device Info (隱含) | `buildNetworkInterfaces()` | discovery.go | 138 |
| 介面角色轉換 | `DiscoveryHandler.parseInterfaceRole()` | discovery.go | 177 |

---

## 資料流向總結

```
Advertiser 端 (Sender)
======================
device.Collector.Collect()
         ↓
   DeviceInfo (領域模型)
         ↓
   DeviceInfo.ToTXTRecords()
         ↓
   TXT Map (廣告層)
         ↓
   zeroconf.Register() 廣告到網路

                        ↕ mDNS 網路傳輸
                        
Scanner 端 (Receiver)
====================
   zeroconf.Browse() 接收
         ↓
   zeroconf.ServiceEntry.Text (字串陣列)
         ↓
   Scanner.parseEntry() 解析
         ↓
   ServiceEntry (TXT Map)
         ↓
   DiscoveryHandler.serviceEntryToEdgeDevice() 轉換
         ↓
   Protobuf EdgeDevice
         ↓
   gRPC DiscoverResponse
         ↓
   JSON 給客戶端
```
