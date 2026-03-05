const API_BASE = '/api';

export const optimizeImage = async (file, options) => {
    const formData = new FormData();
    formData.append('file', file);

    for (const key in options) {
        if (options[key] !== undefined && options[key] !== null) {
            formData.append(key, options[key]);
        }
    }

    const response = await fetch(API_BASE + '/optimize', {
        method: 'POST',
        body: formData,
    });

    if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.error || err.Details || 'Error optimizing image');
    }

    const blob = await response.blob();
    const originalSize = parseInt(response.headers.get('X-Original-Size') || '0', 10);
    const outputSize = parseInt(response.headers.get('X-Output-Size') || '0', 10);
    const savingsPercent = response.headers.get('X-Savings-Percent') || '0';

    const contentDisposition = response.headers.get('Content-Disposition') || '';
    const match = contentDisposition.match(/filename="(.+?)"/);
    const filename = match ? match[1] : 'optimized' + file.name.substring(file.name.lastIndexOf('.'));

    const mimeType = response.headers.get('Content-Type') || blob.type;
    const url = URL.createObjectURL(blob);

    return { blob, filename, originalSize, outputSize, savingsPercent, mimeType, url };
};

export const optimizeSVG = async (file, options) => {
    const formData = new FormData();
    formData.append('file', file);

    for (const key in options) {
        if (options[key] !== undefined && options[key] !== null) {
            formData.append(key, options[key]);
        }
    }

    const response = await fetch(API_BASE + '/optimize/svg', {
        method: 'POST',
        body: formData,
    });

    if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.error || err.Details || 'Error optimizing SVG');
    }

    const blob = await response.blob();
    const originalSize = parseInt(response.headers.get('X-Original-Size') || '0', 10);
    const outputSize = parseInt(response.headers.get('X-Output-Size') || '0', 10);
    const savingsPercent = response.headers.get('X-Savings-Percent') || '0';

    const contentDisposition = response.headers.get('Content-Disposition') || '';
    const match = contentDisposition.match(/filename="(.+?)"/);
    const filename = match ? match[1] : 'optimized' + file.name.substring(file.name.lastIndexOf('.'));

    const mimeType = response.headers.get('Content-Type') || blob.type;
    const url = URL.createObjectURL(blob);

    return { blob, filename, originalSize, outputSize, savingsPercent, mimeType, url };
};

export const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
};

export const isSVG = (file) => {
    return file.type === 'image/svg+xml' || file.name.endsWith('.svg');
};

export const fetchFormats = async () => {
    const res = await fetch(API_BASE + '/formats');
    if (!res.ok) throw new Error('Failed to fetch formats');
    return res.json();
};

export const healthCheck = async () => {
    const res = await fetch(API_BASE + '/health');
    if (!res.ok) throw new Error('Health check failed');
    return res.json();
};
