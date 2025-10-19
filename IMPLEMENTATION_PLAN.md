# HTTP Server + Binbag Implementation Plan

## 🎯 Objetivo

Adicionar suporte a HTTP file server para acelerar transferências, com suporte a pivot e modo speed/stealth configurável em runtime.

## 📊 Status Atual

- ✅ Planejamento completo
- ✅ Documentação criada
- ⏳ Implementação (continuar na próxima sessão)

## 🗂️ Arquivos a Criar/Modificar

### CRIAR:
1. `internal/config.go` - TOML config loading/saving
2. `internal/runtime_config.go` - Thread-safe runtime configuration
3. `internal/fileserver.go` - HTTP server para servir binbag

### MODIFICAR:
4. `internal/transfer.go` - Add SmartUpload with HTTP + fallback
5. `internal/session.go` - Add RunXFromDisk methods + config/set commands
6. `internal/modules.go` - Check runtime config before execution
7. `main.go` - Initialize HTTP server on startup
8. `go.mod` - Add dependency: github.com/BurntSushi/toml

## 🔧 Dependência Nova

```bash
go get github.com/BurntSushi/toml
```

## 📝 Config File Format

```toml
# ~/.gummy/config.toml (SIMPLIFICADO!)

[binbag]
enabled = true
path = "/home/chsoares/Lab/binbag"
http_port = 8080

[execution]
default_mode = "stealth"  # ou "speed"

[pivot]
enabled = false
host = ""
port = 0
```

## 🎯 Lógica Automática (Não Configurável)

**Transfer:**
- Binbag enabled? → Try HTTP first, fallback to b64 chunks (automático)
- Binbag disabled? → Use b64 chunks always

**Cleanup:**
- Disk operations → SEMPRE faz shred (não configurável)

**Módulos:**
- In-memory capable (peas, lse, ps1, net, py, sh):
  - Stealth mode → In-memory (lento, stealth)
  - Speed mode → Disk+cleanup (rápido)
- Disk-only (pspy) → SEMPRE disk+cleanup (modo não importa)
- Bulk upload (privesc) → SEMPRE disk-only (modo não importa)

## 🎮 Comandos do Usuário

```bash
# Ver config
config

# Salvar config
config save

# Alternar modo
set execution speed
set execution stealth

# Configurar binbag
set binbag /path/to/binbag
set binbag off

# Configurar pivot
set pivot 172.16.20.2:8080
set pivot off
```

## 🔄 Fluxo de Upload (Smart)

1. **Resolve Source:**
   - Binbag? Use arquivo existente
   - URL? Download como `tmp_*` no binbag
   - Local path? Copy como `tmp_*` no binbag

2. **Transfer:**
   - Binbag enabled? Try HTTP first
   - HTTP timeout/error? Fallback to b64 chunks
   - No binbag? Use b64 chunks directly

3. **Cleanup:**
   - Remove `tmp_*` files após transfer
   - Shred files on victim if mode=speed

## 🎯 Pivot Support

Quando pivot configurado:
```go
// Sem pivot
url = "http://10.10.14.5:8080/winpeas.exe"

// Com pivot (set pivot 172.16.20.2:8080)
url = "http://172.16.20.2:8080/winpeas.exe"
```

## 🔐 Modos de Execução

### Stealth Mode (padrão)
- Modules in-memory usam b64 chunks (lento, sem disk)
- `run net` → 5 minutos, zero artifacts
- Símbolo: 💾

### Speed Mode (force_disk_execution=true)
- Modules in-memory viram disk+cleanup (rápido, shred depois)
- `run net` → 2 segundos via HTTP, shred após
- Símbolo: 🧹

## 🔧 Decisões de Design (FINALIZADAS)

### 1. Config Simplificado
- ✅ Apenas 3 seções: binbag, execution, pivot
- ✅ Removido: transfer.*, modules.* (tudo automático agora)
- ✅ Cleanup SEMPRE ativo (não configurável)
- ✅ Fallback SEMPRE ativo (não configurável)

### 2. Função Unificada RunFromDisk
- ✅ **UMA função** ao invés de 5 (RunScript, RunBinary, RunDotNetFromDisk, etc)
- ✅ Funciona para qualquer tipo: binários, assemblies, scripts
- ✅ SEMPRE faz cleanup com shred
- ✅ Platform-aware (Windows vs Linux)

### 3. Módulos Simplificados
```go
// Módulos com suporte in-memory
if mode == "speed" {
    RunFromDisk()    // Universal!
} else {
    RunXInMemory()   // Específico
}

// Módulos sem in-memory (pspy, privesc)
RunFromDisk()        // Sempre, modo não importa
```

## 📋 Checklist de Implementação

### Fase 1: Config & HTTP Server
- [ ] `config.go` - Load/save TOML
- [ ] `runtime_config.go` - Thread-safe operations
- [ ] `fileserver.go` - HTTP server
- [ ] Test: Start server, serve files

### Fase 2: Smart Transfer
- [ ] `transfer.go` - Add `SmartUpload()`
- [ ] `transfer.go` - Add `resolveSource()`
- [ ] `transfer.go` - Add `tryHTTPDownload()`
- [ ] Test: Upload with HTTP + fallback

### Fase 3: Unified Disk Execution (SIMPLIFICADO!)
- [ ] `session.go` - `RunFromDisk()` - UMA função universal
- [ ] Remove old functions: RunScript, RunBinary (deprecated)
- [ ] Add platform detection (Windows vs Linux execution)
- [ ] Test: Execute .exe, .ps1, .sh, binaries com cleanup

### Fase 4: Module Integration
- [ ] `modules.go` - Check `GlobalRuntimeConfig.GetMode()`
- [ ] All modules - Switch between in-memory and disk
- [ ] Test: `set execution speed` then `run net`

### Fase 5: Commands
- [ ] `session.go` - Add `config` command
- [ ] `session.go` - Add `set` command handler
- [ ] `session.go` - Update `modules` output
- [ ] Test: All commands work

### Fase 6: Polish
- [ ] Update `modules` table with mode indicator
- [ ] Show execution mode in run output
- [ ] Add pivot support in HTTP URLs
- [ ] Test: Full workflow with pivot

## 🚀 Exemplo de Uso Final

```bash
# Setup
󰗣 gummy ❯ set binbag ~/Lab/binbag
 Binbag enabled: /home/user/Lab/binbag
 HTTP server started on 0.0.0.0:8080
 Found 15 files in binbag

# Configurar pivot (rede interna)
󰗣 gummy [1] ❯ set pivot 172.16.20.2:8080
 Pivot configured: 172.16.20.2:8080

# Modo speed para execução rápida
󰗣 gummy [1] ❯ set execution speed
 Switched to ⚡ speed mode (disk+cleanup)

# Executar módulo (agora rápido!)
󰗣 gummy [1] ❯ run net SharpUp.exe audit
 Running module: net (custom)
 Mode: 🧹 Disk execution with cleanup (speed)
 Using cached SharpUp.exe
 Uploaded via HTTP (2 seconds)
 Executing SharpUp.exe from disk...
 Cleaning up (shred -vfz)...
 Output saved to: SharpUp-2025_10_19-15_30_00-output.txt
```

## ⚠️ Notas Importantes

1. **Pivot não melhora in-memory** - Ainda precisa b64 chunks para variável
2. **HTTP só ajuda disk operations** - 100x mais rápido que chunks
3. **Fallback sempre disponível** - Se HTTP falhar, usa b64 automático
4. **Thread-safe** - RuntimeConfig usa mutex para concorrência
5. **Backward compatible** - Tudo opcional, padrão = comportamento atual

## 🐛 Possíveis Problemas

1. **HTTP timeout em pivot** - Solução: Fallback to chunks
2. **Binbag path inválido** - Solução: Validar antes de habilitar
3. **Porta já em uso** - Solução: Error message, não crashar
4. **Arquivo não encontrado** - Solução: Check 3 sources (binbag, URL, local)

## 📚 Próximos Passos (Sessão Futura)

1. Adicionar dependência TOML
2. Criar `config.go` completo
3. Criar `runtime_config.go` completo
4. Criar `fileserver.go` completo
5. Testar HTTP server standalone
6. Continuar com transfer.go...

---

**TODO:** Ler este arquivo na próxima sessão para continuar implementação!
