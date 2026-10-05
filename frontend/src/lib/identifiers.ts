// Mirrors validNPI and taxonomyPattern in internal/services/claim_x12.go so staff see a bad
// identifier when they enter it, not when a claim fails to submit.
export function isValidNpi(npi: string): boolean {
  // Every NPI CMS issues starts with 1 or 2.
  if (!/^[12]\d{9}$/.test(npi)) return false;
  let sum = 24; // the 80840 prefix's contribution to the Luhn sum
  for (let i = 0; i < 9; i++) {
    let d = Number(npi[8 - i]);
    if (i % 2 === 0) {
      d *= 2;
      if (d > 9) d -= 9;
    }
    sum += d;
  }
  return (10 - (sum % 10)) % 10 === Number(npi[9]);
}

export function isValidTaxonomyCode(code: string): boolean {
  return /^[0-9A-Z]{9}X$/.test(code);
}
