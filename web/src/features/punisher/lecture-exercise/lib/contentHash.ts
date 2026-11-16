/**
 * The content hash an attempt carries (server services.BlockContentHash): the
 * first 12 hex of sha1(problem + "\0" + solution). Synchronous, so it works
 * outside secure contexts too (crypto.subtle is https/localhost only).
 */
export function blockContentHash(problem: string, solution: string): string {
  return sha1Hex(`${problem}\u0000${solution}`).slice(0, 12);
}

function rotl(n: number, b: number): number {
  return (n << b) | (n >>> (32 - b));
}

export function sha1Hex(text: string): string {
  const bytes = new TextEncoder().encode(text);
  const bitLength = bytes.length * 8;
  const words = new Uint32Array((((bytes.length + 8) >> 6) + 1) * 16);
  bytes.forEach((byte, index) => {
    words[index >> 2] |= byte << (24 - (index % 4) * 8);
  });
  words[bytes.length >> 2] |= 0x80 << (24 - (bytes.length % 4) * 8);
  words[words.length - 1] = bitLength >>> 0;
  words[words.length - 2] = Math.floor(bitLength / 0x100000000);
  let h0 = 0x67452301, h1 = 0xefcdab89, h2 = 0x98badcfe, h3 = 0x10325476, h4 = 0xc3d2e1f0;
  const w = new Uint32Array(80);
  for (let off = 0; off < words.length; off += 16) {
    for (let t = 0; t < 16; t++) { w[t] = words[off + t]; }
    for (let t = 16; t < 80; t++) { w[t] = rotl(w[t - 3] ^ w[t - 8] ^ w[t - 14] ^ w[t - 16], 1); }
    let a = h0, b = h1, c = h2, d = h3, ee = h4;
    for (let t = 0; t < 80; t++) {
      let f: number, k: number;
      if (t < 20) { f = (b & c) | (~b & d); k = 0x5a827999; }
      else if (t < 40) { f = b ^ c ^ d; k = 0x6ed9eba1; }
      else if (t < 60) { f = (b & c) | (b & d) | (c & d); k = 0x8f1bbcdc; }
      else { f = b ^ c ^ d; k = 0xca62c1d6; }
      const tmp = (rotl(a, 5) + f + ee + k + w[t]) >>> 0;
      ee = d; d = c; c = rotl(b, 30) >>> 0; b = a; a = tmp;
    }
    h0 = (h0 + a) >>> 0; h1 = (h1 + b) >>> 0; h2 = (h2 + c) >>> 0; h3 = (h3 + d) >>> 0; h4 = (h4 + ee) >>> 0;
  }
  return [h0, h1, h2, h3, h4].map((h) => h.toString(16).padStart(8, "0")).join("");
}
