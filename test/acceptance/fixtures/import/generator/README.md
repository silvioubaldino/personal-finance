# Gerador dos fixtures de import

Gera, a partir de cada `<id>/document.yaml`, o gabarito da leitura da IA (`extraction.json`)
e os PDFs fictícios da pasta (`document.pdf` e, quando há `password:`, `document-protegido.pdf`).
Fatura usa o layout de fatura de cartão; extrato usa o layout de conta corrente.

Regenerar tudo (a partir da raiz do repo):

    python3 test/acceptance/fixtures/import/generator/generate.py

Requisitos: Python 3 com `pyyaml` e `pypdf` (com `cryptography` para AES-128) e Node com
`playwright` e Chromium instalado.

- O CI nunca executa o gerador: os arquivos gerados são commitados e o CI só os consome.
- Os dados são 100% fictícios (nomes, cartões, valores e datas); não há dado pessoal real.
- O HTML intermediário vai para um diretório temporário e não fica nas pastas dos fixtures.
- `render.js` (`node render.js in.html out.pdf`) renderiza uma página HTML em PDF 446×632pt.
- Editar um `document.yaml` e rodar o gerador atualiza a leitura e o PDF da própria pasta.
