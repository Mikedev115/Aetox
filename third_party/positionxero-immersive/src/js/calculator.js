/**
 * Lead-value calculator.
 *
 * Three inputs the visitor owns — what a customer is worth, how many qualified
 * leads land in a month, what share of those close — and one answer:
 *
 *     annual revenue = worth × leads × closeRate × 12
 *
 * Deliberately plain arithmetic, and the page says so underneath the number.
 * Nothing here is scrubbed by scroll depth: the figure must only ever move
 * because the visitor moved a slider, otherwise the result stops being theirs.
 */

const money = new Intl.NumberFormat('en-US', {
  style: 'currency', currency: 'USD', maximumFractionDigits: 0
});

/** 4.5 reads better than 5 here — it is a rate, not a headcount */
function customersLabel(n) {
  return n >= 10 ? String(Math.round(n)) : String(Math.round(n * 10) / 10);
}

export class LeadCalculator {
  constructor(root = document.querySelector('.calc')) {
    this.root = root;
    if (!root) return;

    this.fields = [
      { input: root.querySelector('#calc-worth'), out: root.querySelector('#calc-worth-out'), fmt: (v) => money.format(v) },
      { input: root.querySelector('#calc-leads'), out: root.querySelector('#calc-leads-out'), fmt: (v) => String(v) },
      { input: root.querySelector('#calc-close'), out: root.querySelector('#calc-close-out'), fmt: (v) => `${v}%` }
    ];
    if (this.fields.some((f) => !f.input)) { this.root = null; return; }

    this.annual = root.querySelector('#calc-annual');
    this.monthly = root.querySelector('#calc-monthly');
    this.customers = root.querySelector('#calc-customers');

    for (const f of this.fields) {
      f.input.addEventListener('input', () => this.render());
    }
    this.render();
  }

  /** paints the filled portion of the track — CSS reads --fill */
  _paint(input) {
    const min = Number(input.min);
    const span = Number(input.max) - min;
    const p = span > 0 ? (Number(input.value) - min) / span : 0;
    input.style.setProperty('--fill', `${(p * 100).toFixed(2)}%`);
  }

  render() {
    if (!this.root) return;

    for (const f of this.fields) {
      const v = Number(f.input.value);
      if (f.out) f.out.textContent = f.fmt(v);
      this._paint(f.input);
    }

    const [worth, leads, close] = this.fields.map((f) => Number(f.input.value));
    const won = leads * (close / 100);
    const perMonth = worth * won;

    if (this.annual) this.annual.textContent = money.format(perMonth * 12);
    if (this.monthly) this.monthly.textContent = money.format(perMonth);
    if (this.customers) this.customers.textContent = customersLabel(won);
  }
}
