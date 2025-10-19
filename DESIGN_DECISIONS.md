# Design Decisions - HTTP Server + Binbag

Este documento registra as decisões importantes tomadas durante o planejamento da feature.

## 🎯 Problema Original

- **Windows uploads muito lentos**: 1KB chunks via b64 = ~67KB/s (5 minutos para 10MB)
- **Penelope usa HTTP server**: Mas para de funcionar em redes pivotadas
- **Binbag desorganizado**: Baixar mesma ferramenta múltiplas vezes

## ✅ Solução Implementada

### 1. HTTP File Server + Binbag
- Servir arquivos de `~/Lab/binbag` via HTTP
- Vítima faz `wget/curl` direto (100x mais rápido)
- Fallback automático para b64 chunks se HTTP falhar

### 2. Config Minimalista

**Decisão:** Apenas 3 configurações necessárias

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

**Rejeitado:**
```toml
[transfer]
prefer_http_for_disk = true    # ❌ Redundante (automático)
fallback_to_chunks = true      # ❌ Redundante (automático)

[modules]
cleanup_after_execution = true # ❌ Deve ser sempre ativo
```

**Razão:** Lógica automática é melhor que configuração explícita.
- HTTP vs chunks: Decide automaticamente (binbag enabled?)
- Fallback: Sempre ativo (HTTP timeout? usa chunks)
- Cleanup: Sempre ativo em disk operations

### 3. Função Unificada RunFromDisk

**Decisão:** UMA função ao invés de 5

```go
// ANTES (5 funções)
RunScript()            // Bash scripts
RunBinary()            // Binários
RunDotNetFromDisk()    // .NET assemblies
RunPowerShellFromDisk() // PowerShell scripts
RunPythonFromDisk()    // Python scripts

// AGORA (1 função)
RunFromDisk()          // TUDO! Universal!
```

**Razão:**
- Comportamento idêntico em todas
- SmartUpload + Execute + Shred cleanup
- Platform-aware (Windows vs Linux)
- Menos código duplicado

### 4. Execução Automática dos Módulos

**Decisão:** Módulos checam `GetMode()` e decidem automaticamente

```go
func (m *DotNetModule) Run() {
    if GlobalRuntimeConfig.GetMode() == "speed" {
        return session.RunFromDisk()  // Rápido
    } else {
        return session.RunInMemory()  // Stealth
    }
}

// Módulos sem in-memory
func (m *PSPYModule) Run() {
    return session.RunFromDisk()  // Sempre
}
```

**Rejeitado:**
- Configuração por módulo (muito complexo)
- Flags --force-disk (verboso demais)
- Detecção automática de tamanho (imprevisível)

**Razão:** Modo global é simples e previsível. User controla com `set execution`.

### 5. Pivot Support

**Decisão:** Config simples, substituição automática de IP

```toml
[pivot]
enabled = true
host = "172.16.20.2"
port = 8080
```

Quando ativo:
```
http://10.10.14.5:8080/winpeas.exe  →  http://172.16.20.2:8080/winpeas.exe
```

**Rejeitado:**
- Autodetecção de pivot (impossível sem conhecer topologia)
- Múltiplos pivots (complexo demais para uso comum)

**Razão:** Solução simples que resolve o problema do Penelope.

### 6. Smart Upload Flow

**Decisão:** Prioridade clara: Binbag > URL > Local Path

```
1. Check binbag: winpeas.exe existe? Use!
2. URL? Download como tmp_winpeas.exe no binbag
3. Local path? Copy como tmp_* no binbag
4. Transfer: HTTP first, fallback to b64 chunks
5. Cleanup: Remove tmp_* files
```

**Rejeitado:**
- Cache em /tmp local (poluição)
- Sem cleanup de tmp_* (poluição do binbag)
- Timestamp em todos os files (dificulta cache)

**Razão:** Binbag vira cache central. Temporários são limpos automaticamente.

## 🎯 Benefícios Finais

### Performance
- ✅ Upload 10MB: 5 minutos → 2 segundos (150x faster)
- ✅ Múltiplos uploads: Cache = instant
- ✅ Binbag compartilhado entre sessões

### UX
- ✅ `set execution speed` → Tudo fica rápido
- ✅ `set pivot 172.16.20.2:8080` → Funciona em redes internas
- ✅ Config simples (3 seções, mínimo de opções)

### Robustez
- ✅ Fallback automático (HTTP falha? usa chunks)
- ✅ Cleanup automático (tmp_* e shred)
- ✅ Platform-aware (Windows vs Linux)

### Simplicidade
- ✅ 1 função ao invés de 5 (RunFromDisk)
- ✅ Lógica automática > config explícita
- ✅ Módulos decidem automaticamente baseado em modo

## ⚠️ Limitações Conhecidas

### In-Memory NÃO melhora com HTTP
- HTTP baixa rápido, mas chunking para variável continua lento
- Solução: Modo speed (disk+cleanup) quando velocidade é prioridade

### Pivot Requer Configuração Manual
- Não há autodetecção de topologia de rede
- User precisa saber IP do pivot

### Binbag Requer Organização
- Path deve existir e ter permissões corretas
- User precisa manter binbag atualizado

## 🔮 Futuro (Não Implementar Agora)

- [ ] Auto-update de binbag (baixar versões novas)
- [ ] Múltiplos binbags (~/Lab/binbag, ~/Tools, etc)
- [ ] Detecção automática de pivot via routing table
- [ ] Cache inteligente com TTL

---

**Decisões tomadas:** 2025-10-19
**Revisão:** Aprovadas pelo usuário
**Status:** Pronto para implementação
