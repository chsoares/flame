# 🚀 Continuar na Próxima Sessão

## 📋 Status Atual

✅ **Completado nesta sessão:**
1. Planejamento completo do HTTP Server + Binbag
2. Documentação detalhada em `IMPLEMENTATION_PLAN.md`
3. Esqueleto criado:
   - `internal/config.go` ✅ (pronto, mas não testado)
   - `internal/runtime_config.go` ✅ (esqueleto com TODOs)
   - `internal/fileserver.go` ✅ (esqueleto com TODOs)

⏳ **Para fazer na próxima sessão:**
4. Implementar TODOs nos arquivos criados
5. Modificar `transfer.go` para SmartUpload
6. Adicionar RunXFromDisk methods em `session.go`
7. Atualizar `modules.go` para checar config
8. Adicionar comandos `config` e `set`
9. Integrar HTTP server no `main.go`

## 🎯 Primeiro Passo na Próxima Sessão

```bash
# 1. Adicionar dependência
cd /home/chsoares/Repos/gummy
go get github.com/BurntSushi/toml

# 2. Ler documentação
cat IMPLEMENTATION_PLAN.md

# 3. Implementar TODOs em ordem:
# - runtime_config.go (completar métodos)
# - fileserver.go (completar Start/Stop)
# - Testar HTTP server standalone
```

## 📚 Arquivos Importantes

**LEIA PRIMEIRO:**
1. **DESIGN_DECISIONS.md** - ⭐ Decisões de design e razões (NOVO!)
2. **IMPLEMENTATION_PLAN.md** - Plano técnico completo

**IMPLEMENTAÇÃO:**
3. **internal/config.go** - Config loading/saving (✅ pronto)
4. **internal/runtime_config.go** - Runtime config (⏳ TODOs)
5. **internal/fileserver.go** - HTTP server (⏳ TODOs)

## 🎯 Config SIMPLIFICADO (Mudança Importante!)

Apenas **3 configurações** necessárias:
```toml
[binbag]
enabled = true
path = "~/Lab/binbag"
http_port = 8080

[execution]
default_mode = "stealth"  # ou "speed"

[pivot]
enabled = false
host = ""
port = 0
```

**Todo o resto é lógica automática hardcoded:**
- HTTP vs chunks → Automático (binbag enabled? try HTTP : use chunks)
- Cleanup → SEMPRE ativo em disk operations
- Fallback → SEMPRE automático (HTTP falha? usa chunks)

## 🔑 Conceitos Chave

### Smart Upload Flow
```
Source → Resolve (binbag/URL/local) → HTTP (fast) or B64 (fallback)
```

### Execution Modes
- **stealth** (padrão): In-memory, b64 chunks, lento, sem disk
- **speed** (opcional): Disk+cleanup, HTTP, rápido, shred depois

### Pivot Support
- Quando configurado, substitui IP do listener nas URLs HTTP
- Resolve problema do Penelope com redes pivotadas

## 🎮 Comandos Implementar

```bash
config              # Print current config
config save         # Save to TOML
set execution speed # Toggle mode
set binbag <path>   # Enable binbag
set pivot <ip:port> # Configure pivot
```

## ⚠️ Cuidados

1. **Thread-safety**: RuntimeConfig usa mutex
2. **Fallback**: Sempre ter b64 chunks como backup
3. **Cleanup**: Remove tmp_* files após transfer
4. **Validation**: Validar paths/ports antes de usar
5. **Backward compat**: Tudo opcional, não quebrar código existente

## 🔑 Decisões Importantes (Discutidas e Aprovadas)

### 1. Config Minimalista
- ✅ Apenas 3 seções (binbag, execution, pivot)
- ❌ SEM transfer.* (tudo automático)
- ❌ SEM cleanup config (sempre ativo)

### 2. RunFromDisk Unificado
- ✅ UMA função para tudo (scripts, binários, assemblies)
- ❌ NÃO ter RunScript, RunBinary, RunDotNetFromDisk separados
- ✅ Platform-aware (Windows vs Linux)
- ✅ SEMPRE faz cleanup

### 3. Módulos Automáticos
- ✅ Checam GetMode() e decidem in-memory vs disk
- ✅ pspy, privesc → SEMPRE disk (modo não importa)
- ❌ NÃO ter config por módulo

## 🧪 Teste Final

Quando tudo estiver pronto:

```bash
# 1. Configurar binbag
set binbag ~/Lab/binbag

# 2. Modo speed
set execution speed

# 3. Testar upload grande
upload ~/Downloads/bigfile.exe

# 4. Executar módulo
run net SharpUp.exe audit

# Resultado esperado:
# - HTTP download em ~2 segundos (não 5 minutos)
# - Execução from disk
# - Shred após execução
# - Output salvo corretamente
```

## 📝 Commits Planejados

Depois de implementar tudo:

1. `feat: add TOML config support`
2. `feat: implement HTTP file server for binbag`
3. `feat: add smart transfer with HTTP fallback`
4. `feat: add speed/stealth execution modes`
5. `feat: add pivot support for internal networks`
6. `feat: add config and set runtime commands`

---

**LEMBRE-SE:** Ler `IMPLEMENTATION_PLAN.md` antes de continuar!
