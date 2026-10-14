// Format exact decimal strings for display without numeric conversion.
/** @param {string} value @returns {string} */
export function formatDecimal(value) {
  const [integer, fraction] = value.split('.');
  const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
  return fraction === undefined ? grouped : `${grouped},${fraction}`;
}

// Display-only half-up rounding; preserve exact input and avoid floating point.
/** @param {string} value @returns {string} */
export function formatChartAmount(value) {
  const negative = value.startsWith('-');
  const [integer, fraction = ''] = (negative ? value.slice(1) : value).split('.');
  let cents = BigInt(integer) * 100n + BigInt(fraction.padEnd(2, '0').slice(0, 2));
  if (fraction.length > 2 && fraction[2] >= '5') cents += 1n;
  const digits = cents.toString().padStart(3, '0');
  const sign = negative && cents !== 0n ? '-' : '';
  return formatDecimal(`${sign}${digits.slice(0, -2)}.${digits.slice(-2)}`);
}
