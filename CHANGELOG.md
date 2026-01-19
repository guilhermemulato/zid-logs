# Changelog

## 0.1.10.17
- Post-rotate passa a ser executado tambem em rotacao agendada.
- Evita envio duplicado de sinal no fluxo de rotacao manual/size-based.

## 0.1.10.10
- Status agora inclui resumo, envio e rotacao, e leitura read-only do state nao faz update de bucket.

## 0.1.10.9.4
- Ajuste do rc.d para usar daemon -f compatível com pfSense.

## 0.1.10.9.3
- Update passa a usar onestart ao reiniciar o servico, evitando ficar parado.

## 0.1.10.9.2
- Start do servico agora fecha stdout/stderr para nao travar update na WebGUI.

## 0.1.10.9.1
- Rotacao agendada agora força rotacao independente do tamanho do arquivo.

## 0.1.10.9
- Rotacao agora ocorre em horario fixo (HH:MM) e envio passa a usar intervalo em horas.

## 0.1.10.8
- Update na WebGUI agora transmite a saida em tempo real e finaliza com retorno.

## 0.1.10.7
- Registro do pacote agora usa a versao do binario para manter a WebGUI atualizada.
- Update passa a alinhar rc.conf.local com o enable do ZID Logs.
- Constante de versao do binario atualizada para refletir o bundle.

## 0.1.10.6
- Versao na WebGUI agora usa config/XML como fonte principal para refletir o update.

## 0.1.10.5
- Versao na WebGUI agora exibe apenas o numero.
- Update reinicia o servico quando estiver habilitado, garantindo status running.

## 0.1.10.4
- Update agora roda em primeiro plano na WebGUI, exibindo saida e evitando travar pela recarga do WebGUI.

## 0.1.10.3
- Alinhado layout das abas Settings, Status e Inputs ao padrao do ZID Geolocation.

## 0.1.10.2
- Update remoto agora roda em background para evitar travamento da WebGUI.

## 0.1.10.1
- Status abre estado em modo somente leitura com timeout para evitar travamento da aba.

## 0.1.10.16
- Notificacao pos-rotacao (signal/command) para reabrir logs.
- Rotacao inteligente por timestamp_layout no horario agendado.

## 0.1.10.13
- Scheduler de rotacao por horario avalia a cada minuto.

## 0.1.10.12
- Rotacao agendada executa no start se horario ja passou.
- Status read-only tolera timeout no state.db (nao quebra WebGUI).

## 0.1.10
- Update movido para Service controls.
- Start automatico ao habilitar e restart apos update.

## 0.1.9
- Controles de servico e status na aba Settings.

## 0.1.8
- Ajustes de labels em ingles, campo de auth header e remocao de TLS na GUI.
- Aba Settings como primeira.

## 0.1.7
- Corrigido estado das abas na WebGUI.

## 0.1.6
- Ajustes de layout WebGUI com estilo padrao pfSense.

## 0.1.5
- Corrigido erro de parse em zid-logs_config.php.

## 0.1.4
- Pacote pfSense registrado via XML/INC e scripts de registro.
- Install/uninstall atualizados para registrar menu na WebGUI.

## 0.1.3
- Adicionados testes unitarios (config, registry, rotate, state, shipper).
- Bundle latest e script de empacotamento para pfSense.
- Documentacao atualizada (README e INSTALL-PFSENSE).

## 0.1.2
- Adicionados scripts de install/update/uninstall e bootstrap updater.
- GUI adicionou acao de update via CLI.
