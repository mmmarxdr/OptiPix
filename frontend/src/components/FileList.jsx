import React from 'react';
import { formatBytes, isSVG } from '../utils/api';

function FileRow({ item, result, onRemove, onDownload }) {
    const { file, status, preview, error } = item;
    const isSvgFile = isSVG(file);

    const originalSize = result?.originalSize || file.size;
    const outputSize = result?.outputSize;
    const savings = result?.savingsPercent ? parseFloat(result.savingsPercent) : 0;

    return (
        <div className={`file-row ${status === 'processing' ? 'file-row--processing' : ''} ${status === 'error' ? 'file-row--error' : ''}`}>
            <div className="file-row__thumb">
                {preview ? (
                    <img src={preview} alt="" />
                ) : (
                    <div className="file-row__thumb-placeholder">{isSvgFile ? 'SVG' : '?'}</div>
                )}
            </div>

            <div className="file-row__info">
                <div className="file-row__name">{file.name}</div>
                <div className="file-row__size">
                    {formatBytes(originalSize)}
                    {outputSize > 0 && (
                        <>
                            <svg className="summary-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                                <polyline points="12 5 19 12 12 19"></polyline>
                            </svg>
                            {formatBytes(outputSize)}
                        </>
                    )}
                </div>
            </div>

            <div className="file-row__status">
                {status === 'processing' && <div className="spinner--sm"></div>}
                {status === 'error' && (
                    <div className="file-row__error" title={error} style={{ color: 'var(--red)', fontSize: '0.85rem', fontWeight: '600', cursor: 'help' }}>
                        Failed
                    </div>
                )}
                {status === 'done' && (
                    <div className={`file-row__savings ${savings > 0 ? 'file-row__savings--positive' : 'file-row__savings--negative'}`}>
                        {savings > 0 ? '−' : '+'}{Math.abs(savings)}%
                    </div>
                )}
            </div>

            <div className="file-row__actions">
                {status === 'done' && (
                    <button className="btn-icon" onClick={() => onDownload(result)} title="Download">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                            <polyline points="7 10 12 15 17 10" />
                            <line x1="12" y1="15" x2="12" y2="3" />
                        </svg>
                    </button>
                )}
                <button className="btn-icon btn-icon--danger" onClick={() => onRemove(item.id)} title="Remove">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                </button>
            </div>
        </div>
    );
}

export default function FileList({ files, results, onRemove, onDownload, onDownloadAll, onClear }) {
    if (files.length === 0) return null;

    const totalResults = results.map(r => r.result);
    const totalOriginal = totalResults.reduce((sum, r) => sum + (r.originalSize || 0), 0);
    const totalOutput = totalResults.reduce((sum, r) => sum + (r.outputSize || 0), 0);
    const totalSavingsPct = totalOriginal > 0 ? ((totalOriginal - totalOutput) / totalOriginal * 100).toFixed(1) : 0;

    return (
        <div className="file-list">
            <div className="file-list__header">
                <h3 className="file-list__title">Files ({files.length})</h3>
                <div className="file-list__actions">
                    {totalResults.length > 0 && (
                        <button className="btn btn--ghost" onClick={onDownloadAll}>Download all</button>
                    )}
                    <button className="btn btn--ghost btn--danger" onClick={onClear}>Clear all</button>
                </div>
            </div>

            {totalResults.length > 0 && (
                <div className="file-list__summary">
                    <div className="summary-stat">
                        <span className="summary-label">Original</span>
                        <span className="summary-value">{formatBytes(totalOriginal)}</span>
                    </div>
                    <svg className="summary-arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <line x1="5" y1="12" x2="19" y2="12"></line>
                        <polyline points="12 5 19 12 12 19"></polyline>
                    </svg>
                    <div className="summary-stat">
                        <span className="summary-label">Optimized</span>
                        <span className="summary-value summary-value--highlight">{formatBytes(totalOutput)}</span>
                    </div>
                    <div className="summary-stat" style={{ marginLeft: 'auto' }}>
                        <span className="summary-value summary-value--savings">−{totalSavingsPct}%</span>
                    </div>
                </div>
            )}

            <div className="file-list__items">
                {files.map(fileItem => {
                    const result = results.find(r => r.id === fileItem.id)?.result;
                    return (
                        <FileRow
                            key={fileItem.id}
                            item={fileItem}
                            result={result}
                            onRemove={onRemove}
                            onDownload={onDownload}
                        />
                    );
                })}
            </div>
        </div>
    );
}
