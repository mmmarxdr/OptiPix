import { useState, useMemo } from 'react';
import { optimizeImage, optimizeSVG, isSVG } from '../utils/api';

export function useOptimizer() {
    const [files, setFiles] = useState([]);
    const [results, setResults] = useState([]);
    const [processing, setProcessing] = useState(false);

    const addFiles = (fileList) => {
        const newFiles = Array.from(fileList).map(file => {
            const isSvgFile = isSVG(file);
            return {
                id: file.name + '-' + Date.now() + '-' + Math.random().toString(36).substring(7),
                file,
                status: 'idle', // idle, processing, done, error
                preview: isSvgFile ? null : URL.createObjectURL(file),
                error: null
            };
        });
        setFiles(prev => [...prev, ...newFiles]);
    };

    const removeFile = (id) => {
        setFiles(prev => {
            const fileToRemove = prev.find(f => f.id === id);
            if (fileToRemove?.preview) {
                URL.revokeObjectURL(fileToRemove.preview);
            }
            return prev.filter(f => f.id !== id);
        });
        setResults(prev => {
            const resultToRemove = prev.find(r => r.id === id);
            if (resultToRemove?.result?.url) {
                URL.revokeObjectURL(resultToRemove.result.url);
            }
            return prev.filter(r => r.id !== id);
        });
    };

    const clearAll = () => {
        files.forEach(f => {
            if (f.preview) URL.revokeObjectURL(f.preview);
        });
        results.forEach(r => {
            if (r.result?.url) URL.revokeObjectURL(r.result.url);
        });
        setFiles([]);
        setResults([]);
    };

    const processAll = async (options) => {
        if (processing) return;
        setProcessing(true);

        const pendingFiles = files.filter(f => f.status === 'idle' || f.status === 'error');

        for (const item of pendingFiles) {
            setFiles(prev => prev.map(f => f.id === item.id ? { ...f, status: 'processing', error: null } : f));

            try {
                const result = isSVG(item.file) ? await optimizeSVG(item.file, { multipass: true, precision: 3 }) : await optimizeImage(item.file, options);

                setResults(prev => [...prev.filter(r => r.id !== item.id), { id: item.id, result }]);
                setFiles(prev => prev.map(f => f.id === item.id ? { ...f, status: 'done' } : f));
            } catch (err) {
                setFiles(prev => prev.map(f => f.id === item.id ? { ...f, status: 'error', error: err.message } : f));
            }
        }

        setProcessing(false);
    };

    const downloadResult = (result) => {
        if (!result?.url) return;
        const a = document.createElement('a');
        a.href = result.url;
        a.download = result.filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    };

    const downloadAll = () => {
        results.forEach(r => downloadResult(r.result));
    };

    const totalSavings = useMemo(() => {
        return results.reduce((acc, r) => {
            acc.original += r.result.originalSize || 0;
            acc.output += r.result.outputSize || 0;
            return acc;
        }, { original: 0, output: 0 });
    }, [results]);

    return {
        files,
        results,
        processing,
        addFiles,
        removeFile,
        clearAll,
        processAll,
        downloadResult,
        downloadAll,
        totalSavings
    };
}
