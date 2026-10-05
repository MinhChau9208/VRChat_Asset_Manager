// Links to pages that show one record. They use query strings instead of
// dynamic segments (/assets/[id]) so the app can be built as a static export
// and embedded in the Go binary.

export const assetHref = (id: number | string) => `/assets?id=${id}`;

export const avatarHref = (id: number | string) => `/avatars?id=${id}`;
