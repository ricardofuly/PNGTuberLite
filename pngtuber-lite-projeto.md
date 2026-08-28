# PNGTuber Lite — Engine leve compatível com avatares do PNGTuber-Plus

> Documento de referência do projeto: visão geral, formato de arquivo, arquitetura, stack recomendada, boas práticas e plano de implementação multiplataforma (Windows + Linux).

Referência original: [kaiakairos/PNGTuber-Plus](https://github.com/kaiakairos/PNGTuber-Plus) (Godot 4.1.2, GDScript).

---

## 1. Objetivo do projeto

Criar uma aplicação nativa, leve (baixo consumo de RAM/CPU, binário pequeno, startup instantâneo), capaz de:

1. Carregar e renderizar arquivos `.save` gerados pelo PNGTuber-Plus original, sem conversão manual.
2. Reproduzir o comportamento visual esperado: hierarquia de camadas, wobble/jiggle idle, piscar (blink), boca sincronizada com áudio (talk), sprite-sheet animado, troca de figurino (costumes).
3. Funcionar como overlay transparente capturável por OBS/streaming software, em Windows e Linux, sem recompilar para cada SO (build cross-platform a partir do mesmo código-fonte).
4. Ter footprint de recursos significativamente menor que uma engine completa (Godot runtime), já que o app faz uma fração do que a engine original faz.

**Não-objetivos (v1):** editor completo de avatar (criar/editar camadas do zero), suporte a Stream Deck, distribuição de novos formatos de avatar incompatíveis com o `.save` original.

---

## 2. Formato do arquivo `.save`

O `.save` é um objeto JSON no nível raiz, mas **não é JSON estritamente válido em todos os campos** — alguns valores são a `repr()` de tipos do Godot (GDScript), serializados como string. Isso é o principal cuidado ao escrever o parser.

### 2.1 Estrutura raiz

```json
{
  "0": { ...camada 0... },
  "1": { ...camada 1... },
  "2": { ...camada 2... }
}
```

- Chaves são strings numéricas (`"0"`, `"1"`, ...) — **não confiar na ordem de iteração**; a ordem de desenho vem do campo `zindex`, não da chave.
- Cada valor é uma "camada" (layer/sprite).

### 2.2 Campos de cada camada

| Campo | Tipo real | Observação |
|---|---|---|
| `identification` | int | ID único da camada (não é a chave do map) |
| `parentId` | int | ID da camada pai — forma uma hierarquia de transform |
| `imageData` | string (base64) | PNG embutido inline — não há dependência de arquivo externo |
| `pos` | string `"Vector2(x, y)"` | **Não é JSON puro**, precisa de parser dedicado |
| `offset` | string `"Vector2(x, y)"` | Idem — offset do pivot/âncora |
| `path` | string | Caminho original (`user://...`), informativo, não usar para I/O |
| `zindex` | int | Ordem de desenho (maior = na frente) |
| `type` | string | Ex.: `"sprite"` |
| `frames` | int | Número de frames do sprite sheet (horizontal) |
| `animSpeed` | float | Velocidade de troca de frame |
| `clipped` | bool | Se o sprite é recortado/clipado |
| `stretchAmount` | float | Fator de esticamento |
| `rLimitMin` / `rLimitMax` | float | Limites de rotação (graus) para o wobble físico |
| `rotDrag` | float | Amortecimento angular (spring-damper) |
| `drag` | float | Amortecimento linear |
| `xAmp` / `xFrq` | float | Amplitude/frequência da oscilação idle no eixo X |
| `yAmp` / `yFrq` | float | Amplitude/frequência da oscilação idle no eixo Y |
| `ignoreBounce` | bool | Desliga a física de bounce para essa camada |
| `showBlink` | int (0/1) | Camada só aparece durante o "piscar" |
| `showTalk` | int (0/1) | Camada só aparece durante "falando" (voice activity) |
| `costumeLayers` | string `"[1, 1, 1, ...]"` | Array serializado como string — visibilidade por slot de figurino (até 10) |

### 2.3 Cuidados no parsing

- **`Vector2(x, y)`**: extrair com regex (`Vector2\(([-\d.]+),\s*([-\d.]+)\)`), não tentar `json.Unmarshal` direto nesse campo.
- **`costumeLayers`**: também é uma string contendo um array — precisa de segundo parse (split por vírgula ou regex) depois do parse JSON principal.
- **`imageData`**: decodificar base64 → PNG → textura da GPU. Fazer isso uma vez no load, nunca por frame.
- **Hierarquia**: montar uma árvore (`parentId` → filhos) antes de renderizar, para aplicar transform acumulado (posição/rotação do pai afeta o filho).
- **Tipos mistos**: alguns campos podem vir como int ou float dependendo de como o Godot serializou — o parser deve ser tolerante (aceitar `json.Number` em Go, ou `Number` genérico em Java/Jackson).

---

## 3. Arquitetura proposta

Módulos desacoplados, para permitir trocar peças (ex.: motor gráfico) sem reescrever o resto:

```
/model      -> structs do avatar + parser do .save (JSON + Vector2 + costumeLayers)
/render     -> desenha camadas respeitando hierarquia (parentId) e zindex
/anim       -> wobble físico (spring-damper), blink timer, stepper de sprite-sheet
/audio      -> captura de microfone + threshold/VAD -> liga/desliga showTalk
/input      -> hotkeys locais (troca de costume, toggle blink manual, etc.)
/window     -> janela transparente, always-on-top, sem borda (overlay para OBS)
/config     -> configurações do usuário (dispositivo de áudio, sensibilidade, atalhos)
```

### 3.1 Pipeline de frame

```
1. Atualiza física idle (spring-damper) por camada  -> dt fixo (ex.: 1/60s)
2. Atualiza timers de blink (randômico, com cooldown)
3. Lê nível de áudio do buffer de captura -> decide showTalk (com hysteresis)
4. Avança frame do sprite-sheet conforme animSpeed
5. Resolve transform final de cada camada (acumulando a partir da raiz da hierarquia)
6. Ordena por zindex e desenha
```

### 3.2 Física do wobble (idle bounce)

Sistema massa-mola simples por camada, não precisa de engine de física completa:

```
angulo_alvo = 0  (repouso)
velocidade_angular += (angulo_alvo - angulo_atual) * rigidez - velocidade_angular * rotDrag
angulo_atual += velocidade_angular * dt
angulo_atual = clamp(angulo_atual, rLimitMin, rLimitMax)
```

Mesma lógica pode se aplicar em X/Y usando `xAmp/xFrq` e `yAmp/yFrq` como uma oscilação senoidal (`amp * sin(2π * frq * t)`) somada ao spring-damper, dependendo de como o original combina os dois — vale validar visualmente comparando com o app original.

### 3.3 Detecção de fala (showTalk)

- Captura contínua do microfone em buffer circular pequeno (ex.: 20–50ms de janela).
- Calcula RMS (root mean square) do buffer.
- Aplica um **threshold com hysteresis** (limiar de entrada mais alto que o de saída) para evitar "flicker" da boca em ruído de fundo.
- Debounce: só alterna estado depois de N frames consecutivos acima/abaixo do limiar, para suavizar.

---

## 4. Stack recomendada

### 4.1 Comparativo direto

| Critério | Go + raylib-go | Java + JavaFX |
|---|---|---|
| Tamanho de binário / footprint | Muito baixo (binário nativo, sem VM) | Maior (precisa de JRE ou runtime via `jlink`) |
| Startup | Quase instantâneo | Alguns ms a mais (JIT warmup) |
| Janela transparente + always-on-top | Suportado nativamente (`rl.FlagWindowTransparent`, `SetWindowState`) | Suportado nativamente (`StageStyle.TRANSPARENT`, `setAlwaysOnTop`) |
| Captura de áudio (microfone) | Sem binding puro-Go maduro; requer cgo (`malgo`/`portaudio`) | Nativo e cross-platform via `javax.sound.sampled.TargetDataLine`, sem dependência externa |
| Click-through de janela (opcional) | Exige chamada nativa via cgo/syscall por SO | Exige JNA/hacks nativos também — nenhum dos dois tem isso pronto |
| Build cross-compile (Win a partir de Linux e vice-versa) | Simples com toolchain de cgo configurada (ou build puro-Go se evitar cgo) | Simples — JAR roda em qualquer SO com JVM; nativos via `jpackage` por plataforma |
| Empacotamento standalone (sem exigir instalação prévia) | Binário único, fácil | Precisa empacotar JRE junto (`jpackage`), aumenta tamanho final |

### 4.2 Recomendação

**Go + raylib-go** como escolha principal, priorizando o requisito de "leve":

- Reaproveita stack já usada em outros projetos (raylib-go).
- Menor footprint de memória e disco — importante para um app que fica rodando em background durante toda a live.
- Trade-off aceito: a captura de áudio via cgo (`gen2brain/malgo`, wrapper de miniaudio) exige mais atenção no build cross-platform (garantir toolchain de C para Windows e Linux), mas é um problema resolvido e bem documentado.

**Alternativa Java + JavaFX** se a prioridade for menor fricção de desenvolvimento (áudio nativo sem cgo) e o footprint maior for aceitável.

### 4.3 Bibliotecas sugeridas (trilha Go)

- Render/janela: `github.com/gen2brain/raylib-go/raylib`
- Áudio (captura): `github.com/gen2brain/malgo`
- JSON: `encoding/json` da stdlib (suficiente, sem necessidade de libs externas)
- Config: `encoding/json` ou `gopkg.in/yaml.v3` para um arquivo de config separado do `.save`
- Hotkeys globais (opcional): `golang.design/x/hotkey` (multiplataforma)

---

## 5. Boas práticas de implementação

### 5.1 Gestão de recursos gráficos

- Decodificar `imageData` (base64 → PNG → textura GPU) **uma vez no load**, nunca por frame.
- Usar **texture atlas** se o avatar tiver muitas camadas pequenas, reduzindo trocas de textura (draw calls) — opcional na v1, mas vale medir antes de otimizar.
- Liberar texturas (`UnloadTexture`) explicitamente ao trocar de avatar, evitando vazamento de VRAM — Go/raylib não faz GC de recursos de GPU automaticamente.

### 5.2 Timestep fixo para física/animação

- Rodar a física do wobble e os timers de blink em um passo fixo (ex.: 60Hz), desacoplado do framerate de renderização, para o comportamento não variar entre monitores de 60Hz e 144Hz.
- Padrão "fixed update + interpolação" evita jitter visual em telas de alta taxa de atualização.

### 5.3 Áudio em thread separada

- Captura de microfone deve rodar em goroutine própria com um buffer lock-free ou canal (`chan`) pequeno, nunca bloqueando o loop de render.
- Calcular RMS fora da hot path de renderização; só o resultado (booleano `isTalking`) cruza para a thread principal.

### 5.4 Parsing defensivo

- Validar campos obrigatórios (`identification`, `imageData`, `pos`) e falhar com mensagem clara se o `.save` estiver corrompido ou vier de uma versão futura do PNGTuber-Plus com campos novos — ignorar campos desconhecidos em vez de quebrar (forward compatibility).
- Cachear o parse do `.save` em memória; não reprocessar base64 a cada troca de costume (isso já é só uma troca de visibilidade, não de imagem).

### 5.5 Separação UI de configuração x overlay

- Duas janelas distintas: uma "janela de controle" (normal, com decoração, onde o usuário troca avatar/costume/config) e a "janela overlay" (transparente, sem borda, capturada pelo OBS). Rodá-las como duas `Window`/contextos de render evita que a UI de configuração apareça na captura.

### 5.6 Testes de regressão visual

- Manter um conjunto de `.save` de teste (incluindo o `defaultAvatar.save`) e comparar renderização pixel-a-pixel (ou por hash perceptual) contra screenshots de referência tiradas do PNGTuber-Plus original, para pegar regressões de posicionamento/rotação cedo.

---

## 6. Otimização de recursos multiplataforma

### 6.1 Geral

- Compilar com flags de otimização de tamanho (`-ldflags="-s -w"` em Go remove símbolos de debug do binário final).
- Evitar alocações por frame no loop principal (reusar buffers/slices já alocados) para não pressionar o GC do Go durante a live.
- Renderizar apenas quando necessário: se a janela estiver oculta ou minimizada, pausar o loop de captura de áudio/física para economizar CPU.

### 6.2 Windows

- Usar `SetWindowLong`/`SetLayeredWindowAttributes` (via `syscall`/cgo) se for necessário click-through — não é obrigatório para o caso de uso de overlay capturado por OBS (Window Capture já lida bem com janelas transparentes sem precisar de click-through).
- Testar em Windows com DPI scaling diferente de 100% — Vector2/pos do `.save` são coordenadas lógicas do Godot, então a conversão para pixels da tela precisa considerar o scale factor.
- Empacotar como executável único (`go build`), sem instalador obrigatório; assinar o binário (code signing) é recomendável para evitar SmartScreen, mas é opcional na v1.

### 6.3 Linux

- Testar em pelo menos X11 e Wayland — comportamento de janela transparente/always-on-top **difere entre os dois**, principalmente em compositores Wayland mais restritivos (alguns não permitem always-on-top sem protocolo específico).
- Considerar fallback: se Wayland não suportar o recurso necessário, documentar rodar via XWayland.
- Testar em pelo menos duas distros/compositores diferentes (ex.: GNOME + KDE) antes de considerar "suportado".

### 6.4 Build cross-platform a partir de uma única base de código

- Se usar cgo (malgo, bindings nativos), configurar toolchains cruzadas (`mingw-w64` para gerar `.exe` a partir de Linux, ou usar CI com runners nativos por SO — mais confiável que cross-compile de cgo).
- Recomendado: pipeline de CI com matriz `windows-latest` / `ubuntu-latest`, cada um compilando nativamente para seu próprio SO, publicando os binários como artefatos de release.

---

## 7. Plano de implementação incremental

1. **Parser + modelo de dados**: ler `.save`, validar contra o `defaultAvatar.save` de exemplo, montar hierarquia de camadas.
2. **Renderer estático**: desenhar as camadas em repouso (sem animação), comparar visualmente com o Godot original.
3. **Sprite-sheet**: suporte a `frames`/`animSpeed` para camadas animadas.
4. **Blink**: timer randômico + toggle de `showBlink`.
5. **Wobble físico**: spring-damper em posição/rotação, respeitando `rLimitMin/Max`, `drag`, `rotDrag`, `ignoreBounce`.
6. **Costumes**: troca de figurino via `costumeLayers`, com hotkey local.
7. **Áudio/talk detection**: captura de mic, RMS + hysteresis, toggle de `showTalk`.
8. **Janela overlay**: transparente, sem borda, always-on-top, testada em Windows e Linux (X11 + Wayland).
9. **Empacotamento**: build único por plataforma, CI com matriz Windows/Linux, testes de regressão visual automatizados.

---

## 8. Riscos e pontos de atenção

- **Licença**: verificar os termos do repositório original antes de distribuir uma implementação "compatível com os arquivos" publicamente.
- **Campos futuros**: o PNGTuber-Plus pode evoluir o formato `.save` (ex.: Stream Deck, novos parâmetros de física) — o parser deve ignorar campos desconhecidos sem quebrar.
- **Wayland**: maior incerteza de compatibilidade para always-on-top/transparência — validar cedo no projeto, não deixar para o final.
- **Precisão do wobble**: sem acesso ao código-fonte exato da física original, replicar o "feel" pode exigir ajuste fino por comparação visual/tentativa e erro.
