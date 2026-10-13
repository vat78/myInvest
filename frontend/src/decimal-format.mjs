// Format exact decimal strings for display without numeric conversion.
/** @param {string} value @returns {string} */
export function formatDecimal(value) {
  const [integer, fraction] = value.split('.');
  const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
  return fraction === undefined ? grouped : `${grouped},${fraction}`;
}
