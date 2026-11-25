#!/usr/bin/env python3
"""Gera os fixtures de import a partir de cada <id>/document.yaml.

Para cada pasta de test/acceptance/fixtures/import/<id>/ produz:
  - extraction.json          leitura "perfeita" da IA (schema de domain.StatementExtractResult)
  - document.pdf             PDF fictício (fatura ou extrato) com /Subject pf-fixture:<id>
  - document-protegido.pdf   só se o yaml tem `password:` (AES-128, mesmo conteúdo)

Uso (a partir da raiz do repo):
  python3 test/acceptance/fixtures/import/generator/generate.py

Dados 100% fictícios. O HTML intermediário vai para um diretório temporário.
"""

import html
import json
import subprocess
import tempfile
import datetime as dt
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path

import yaml
from pypdf import PdfReader, PdfWriter

GENERATOR_DIR = Path(__file__).resolve().parent
IMPORT_DIR = GENERATOR_DIR.parent
RENDER_JS = GENERATOR_DIR / "render.js"

FIXTURE_PREFIX = "pf-fixture:"
PDF_DATE = "D:20260101000000Z"  # data fixa: PDFs reprodutíveis entre execuções
MONTHS = ["jan.", "fev.", "mar.", "abr.", "mai.", "jun.",
          "jul.", "ago.", "set.", "out.", "nov.", "dez."]

CONF_INVOICE = 0.97
CONF_STATEMENT = 0.96



def D(value) -> Decimal:
    return Decimal(str(value))


def money(value) -> float:
    """Arredonda para 2 casas e devolve float (JSON sem -0.0)."""
    q = D(value).quantize(Decimal("0.01"), ROUND_HALF_UP)
    return float(q) + 0.0


def to_date(value) -> dt.date:
    if isinstance(value, dt.datetime):
        return value.date()
    if isinstance(value, dt.date):
        return value
    return dt.date.fromisoformat(str(value))


def brl_abs(value) -> str:
    """R$ 1.650,08 (sem sinal)."""
    q = abs(D(value)).quantize(Decimal("0.01"), ROUND_HALF_UP)
    return "R$ " + pt_number(q)


def pt_number(q: Decimal) -> str:
    s = f"{q:,.2f}"  # 1,650.08
    return s.replace(",", "\u0000").replace(".", ",").replace("\u0000", ".")


def brl_signed(value) -> str:
    """-R$ 2.100,00 / R$ 6.500,00 (extrato)."""
    return ("-" if D(value) < 0 else "") + brl_abs(value)


def brl_invoice_value(value) -> str:
    """Compras sem sinal; pagamentos e estornos com '+'."""
    if D(value) > 0:
        return "+ " + brl_abs(value)
    return brl_abs(value)


def br_date(d: dt.date) -> str:
    return d.strftime("%d/%m/%Y")


def long_date(d: dt.date) -> str:
    return f"{d.day:02d} de {MONTHS[d.month - 1]} {d.year}"


def installment_suffix(item) -> str:
    inst = item.get("installment")
    if not inst:
        return ""
    n, t = inst
    return f" (Parcela {n:02d} de {t:02d})"


def is_payment(description: str) -> bool:
    return description.startswith("PAGAMENTO")



def invoice_movements(doc):
    """Movimentos (ordem do yaml: cartão 1, depois cartão 2) e os itens por cartão."""
    movements, per_card = [], []
    for card in doc["cards"]:
        card_items = []
        for item in card["items"]:
            desc = item["description"] + installment_suffix(item)
            m = {
                "date": to_date(item["date"]).isoformat(),
                "description": desc,
                "amount": money(item["amount"]),
                "type_payment": "credit_card",
            }
            if item.get("installment"):
                n, t = item["installment"]
                m["installment_number"] = n
                m["total_installments"] = t
            movements.append(m)
            card_items.append((desc, item))
        per_card.append((card["number"], card_items))
    return movements, per_card


def invoice_total(movements) -> float:
    total = sum((D(m["amount"]) for m in movements
                 if not is_payment(m["description"])), Decimal("0"))
    return money(total)


def invoice_reading(doc):
    movements, _ = invoice_movements(doc)
    return {
        "document_type": "invoice",
        "confidence": CONF_INVOICE,
        "invoice_meta": {
            "closing_date": to_date(doc["closing_date"]).isoformat(),
            "due_date": to_date(doc["due_date"]).isoformat(),
            "total_amount": invoice_total(movements),
        },
        "movements": movements,
    }


CSS_COMMON = """
@page { size: 446pt 632pt; margin: 0; }
* { box-sizing: border-box; }
html, body { margin: 0; padding: 0; background: #ffffff; }
body { font-family: Inter, "Helvetica Neue", Arial, sans-serif; color: #1d1d1f; }
.page { width: 446pt; height: 632pt; position: relative; overflow: hidden;
        padding: 26px 30px 0; page-break-after: always; }
.page:last-child { page-break-after: auto; }
.label { font-size: 8.5px; color: #6b6b70; margin-bottom: 2px; }
.muted { font-size: 8.5px; color: #6b6b70; }
h1 { font-size: 17px; font-weight: 600; margin: 0 0 14px; }
h2 { font-size: 13px; font-weight: 600; margin: 0 0 12px; color: #1d1d1f; }
.kicker { font-size: 8.5px; font-weight: 600; color: #FF7A00; letter-spacing: .04em;
          text-transform: uppercase; margin-bottom: 6px; }
.bar { height: 4px; background: #FF7A00; border-radius: 2px; margin: 0 -30px 22px; }
table { width: 100%; border-collapse: collapse; font-size: 9px; }
th { text-align: left; font-weight: 600; color: #6b6b70; font-size: 8px;
     padding: 5px 4px; border-bottom: 1px solid #e4e4e8; }
td { padding: 5px 4px; border-bottom: 1px solid #f0f0f3; vertical-align: top; }
.num { text-align: right; white-space: nowrap; }
.card-title { font-size: 9px; font-weight: 600; color: #FF7A00; margin: 14px 0 6px; }
.card-total { text-align: right; font-size: 9px; font-weight: 600; margin-top: 6px; }
footer { position: absolute; left: 30px; right: 30px; bottom: 16px; border-top: 1px solid #e4e4e8;
         padding-top: 7px; font-size: 7.5px; color: #6b6b70; display: flex;
         justify-content: space-between; gap: 8px; }
"""


def invoice_html(doc, movements, per_card) -> str:
    holder = doc["holder"]
    first_name = holder.split()[0].capitalize()
    first_card = doc["cards"][0]["number"]
    due = to_date(doc["due_date"])
    total_abs = brl_abs(invoice_total(movements))
    footer = (
        '<footer><span>{h}</span><span>{c}</span>'
        '<span>Vencimento {d}</span><span>Total {t}</span></footer>'
    ).format(h=html.escape(holder), c=html.escape(first_card),
             d=br_date(due), t=html.escape(total_abs))

    page1 = f"""
<section class="page">
  <div class="bar"></div>
  <div class="kicker">Resumo da fatura</div>
  <h1>Olá, {html.escape(first_name)}! A sua fatura chegou!</h1>
  <div style="margin: 18px 0 22px;">
    <div class="label">Total da sua fatura</div>
    <div style="font-size: 28px; font-weight: 700; color: #1d1d1f;">{html.escape(total_abs)}</div>
    <div class="muted" style="margin-top: 4px;">Este é o valor que você precisa pagar nesse mês</div>
  </div>
  <table style="width: 100%; margin-bottom: 22px;">
    <tr>
      <td style="border: none; padding: 0 0 4px 0;">
        <div class="label">Data de Vencimento</div>
        <div style="font-size: 12px; font-weight: 600;">{br_date(due)}</div>
      </td>
      <td style="border: none; padding: 0 0 4px 0;">
        <div class="label">Limite de crédito total</div>
        <div style="font-size: 12px; font-weight: 600;">{brl_abs(doc["credit_limit"])}</div>
      </td>
    </tr>
  </table>
  <div style="border: 1px solid #e4e4e8; border-radius: 6px; padding: 10px 12px;">
    <div class="label">Pagamento via boleto</div>
    <div style="font-family: 'DejaVu Sans Mono', monospace; font-size: 9.5px;">
      00000.00000 00000.000000 00000.000000 0 00000000000000
    </div>
  </div>
  {footer}
</section>"""

    blocks = []
    for number, items in per_card:
        rows = []
        card_sum = Decimal("0")
        for (desc, item) in items:
            amount = D(item["amount"])
            d = to_date(item["date"])
            rows.append(
                "<tr>"
                f"<td>{html.escape(long_date(d))}</td>"
                f"<td>{html.escape(desc)}</td>"
                "<td>-</td>"
                f'<td class="num">{html.escape(brl_invoice_value(amount))}</td>'
                "</tr>"
            )
            if not is_payment(item["description"]):
                card_sum += amount
        blocks.append(
            f'<div class="card-title">CARTÃO {html.escape(number)}</div>'
            "<table><thead><tr><th style='width:82px'>Data</th><th>Movimentação</th>"
            "<th style='width:70px'>Beneficiário</th><th class='num' style='width:80px'>Valor</th>"
            "</tr></thead><tbody>" + "".join(rows) + "</tbody></table>"
            f'<div class="card-total">Total CARTÃO {html.escape(number)}&nbsp;&nbsp;'
            f"{html.escape(brl_abs(card_sum))}</div>"
        )
    page2 = f"""
<section class="page">
  <div class="bar"></div>
  <h2>Despesas da fatura</h2>
  {''.join(blocks)}
  {footer}
</section>"""

    next_rows = []
    for it in doc.get("next_invoice_installments", []):
        n, t = it["installment"]
        next_rows.append(
            "<tr>"
            f"<td>{html.escape(it['description'])} (Parcela {n:02d} de {t:02d})</td>"
            f'<td class="num">{html.escape(brl_abs(it["amount"]))}</td>'
            "</tr>"
        )
    page3 = f"""
<section class="page">
  <div class="bar"></div>
  <h2>Próxima fatura</h2>
  <div class="muted" style="font-size: 10px; margin-bottom: 14px;">
    Data de corte: {br_date(to_date(doc["next_closing_date"]))}
  </div>
  <table>
    <thead><tr><th>Movimentação</th><th class="num" style="width:100px">Valor</th></tr></thead>
    <tbody>{''.join(next_rows)}</tbody>
  </table>
  {footer}
</section>"""

    return _document(page1 + page2 + page3)



def statement_reading(doc):
    movements = []
    for line in doc["lines"]:
        m = {
            "date": to_date(line["date"]).isoformat(),
            "description": line["description"],
            "amount": money(line["amount"]),
        }
        if line.get("type_payment"):
            m["type_payment"] = line["type_payment"]
        movements.append(m)
    return {
        "document_type": "statement",
        "confidence": CONF_STATEMENT,
        "movements": movements,
    }


def statement_html(doc) -> str:
    balance = D(doc["opening_balance"])
    rows = [
        "<tr><td>{}</td><td>Saldo anterior</td><td></td><td class='num'>{}</td></tr>".format(
            br_date(to_date(doc["period_start"])), html.escape(brl_signed(balance)))
    ]
    for line in doc["lines"]:
        balance += D(line["amount"])
        rows.append(
            "<tr>"
            f"<td>{html.escape(br_date(to_date(line['date'])))}</td>"
            f"<td>{html.escape(line['description'])}</td>"
            f"<td class='num'>{html.escape(brl_signed(line['amount']))}</td>"
            f"<td class='num'>{html.escape(brl_signed(balance))}</td>"
            "</tr>"
        )
    final = html.escape(brl_signed(balance))
    footer = (
        '<footer><span>{b}</span><span>{h}</span><span>{a}</span>'
        '<span>Página 1 de 1</span></footer>'
    ).format(b=html.escape(doc["bank"]), h=html.escape(doc["holder"]),
             a=html.escape(doc["account"]))

    page = f"""
<section class="page">
  <div class="bar"></div>
  <div class="kicker">{html.escape(doc["bank"])}</div>
  <h1>Extrato de conta corrente</h1>
  <table style="margin-bottom: 16px;">
    <tr>
      <td style="border:none; padding: 0 0 4px 0;"><div class="label">Titular</div>
        <div style="font-weight:600;">{html.escape(doc["holder"])}</div></td>
      <td style="border:none; padding: 0 0 4px 0;"><div class="label">Conta</div>
        <div style="font-weight:600;">{html.escape(doc["account"])}</div></td>
    </tr>
    <tr>
      <td style="border:none; padding: 0;"><div class="label">Período</div>
        <div style="font-weight:600;">{br_date(to_date(doc["period_start"]))} a {br_date(to_date(doc["period_end"]))}</div></td>
      <td style="border:none; padding: 0;"></td>
    </tr>
  </table>
  <table>
    <thead><tr>
      <th style="width:62px">Data</th><th>Descrição</th>
      <th class="num" style="width:88px">Valor</th><th class="num" style="width:96px">Saldo</th>
    </tr></thead>
    <tbody>{''.join(rows)}</tbody>
  </table>
  <div style="margin-top: 14px; text-align: right; font-size: 11px; font-weight: 600;">
    Saldo final&nbsp;&nbsp;{final}
  </div>
  {footer}
</section>"""
    return _document(page)



def _document(body: str) -> str:
    return (
        "<!doctype html><html lang='pt-BR'><head><meta charset='utf-8'>"
        f"<style>{CSS_COMMON}</style></head><body>{body}</body></html>"
    )


def render_pdf(html_text: str, workdir: Path, name: str) -> Path:
    html_path = workdir / f"{name}.html"
    pdf_path = workdir / f"{name}.raw.pdf"
    html_path.write_text(html_text, encoding="utf-8")
    subprocess.run(["node", str(RENDER_JS), str(html_path), str(pdf_path)], check=True)
    return pdf_path


def write_pdf(src: Path, dst: Path, fixture_id: str, password: str | None = None) -> None:
    writer = PdfWriter(clone_from=PdfReader(str(src)))
    writer.add_metadata({
        "/Title": fixture_id,
        "/Subject": FIXTURE_PREFIX + fixture_id,
        "/Producer": "personal-finance fixture generator",
        "/CreationDate": PDF_DATE,
        "/ModDate": PDF_DATE,
    })
    if password is not None:
        writer.encrypt(user_password=password, owner_password=password, algorithm="AES-128")
    with open(dst, "wb") as fh:
        writer.write(fh)



def build(folder: Path, workdir: Path) -> None:
    fixture_id = folder.name
    doc = yaml.safe_load((folder / "document.yaml").read_text(encoding="utf-8"))
    if doc.get("id") != fixture_id:
        raise SystemExit(f"{folder}: id no yaml ({doc.get('id')!r}) difere da pasta")

    kind = doc["kind"]
    if kind == "invoice":
        movements, per_card = invoice_movements(doc)
        reading = invoice_reading(doc)
        html_text = invoice_html(doc, movements, per_card)
    elif kind == "statement":
        reading = statement_reading(doc)
        html_text = statement_html(doc)
    else:
        raise SystemExit(f"{folder}: kind desconhecido {kind!r}")

    (folder / "extraction.json").write_text(
        json.dumps(reading, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

    raw = render_pdf(html_text, workdir, fixture_id)
    write_pdf(raw, folder / "document.pdf", fixture_id)

    protected = folder / "document-protegido.pdf"
    if doc.get("password") is not None:
        write_pdf(raw, protected, fixture_id, password=str(doc["password"]))
    elif protected.exists():
        protected.unlink()

    print(f"ok  {fixture_id}")


def main() -> None:
    folders = sorted(
        p for p in IMPORT_DIR.iterdir()
        if p.is_dir() and p.name != "generator" and (p / "document.yaml").exists()
    )
    with tempfile.TemporaryDirectory(prefix="pf-fixtures-") as tmp:
        for folder in folders:
            build(folder, Path(tmp))


if __name__ == "__main__":
    main()
