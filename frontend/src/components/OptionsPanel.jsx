import React, { useState } from 'react';

export default function OptionsPanel({ options, onChange, fileCount, onProcess, processing }) {
    const [expanded, setExpanded] = useState(false);

    const handleFormatChange = (format) => {
        onChange({ ...options, format });
    };

    const handleKeyDown = (e, format) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handleFormatChange(format);
        }
    };

    return (
        <div className="options-panel">
            <div className="options-panel__header">
                <h3 className="options-panel__title">Output Settings</h3>
                <button className="options-panel__toggle btn btn--ghost" onClick={() => setExpanded(!expanded)}>
                    {expanded ? 'Less options' : 'More options'}
                </button>
            </div>

            <div className="options-panel__formats">
                <div role="button" tabIndex={0} className={`format-chip ${options.format === 'webp' ? 'format-chip--active' : ''}`} onClick={() => handleFormatChange('webp')} onKeyDown={(e) => handleKeyDown(e, 'webp')}>
                    <div className="format-chip__label">WebP</div>
                    <div className="format-chip__desc">Best balance of size & quality</div>
                </div>
                <div role="button" tabIndex={0} className={`format-chip ${options.format === 'avif' ? 'format-chip--active' : ''}`} onClick={() => handleFormatChange('avif')} onKeyDown={(e) => handleKeyDown(e, 'avif')}>
                    <div className="format-chip__label">AVIF</div>
                    <div className="format-chip__desc">Smallest size, slower encoding</div>
                </div>
                <div role="button" tabIndex={0} className={`format-chip ${options.format === 'jpeg' ? 'format-chip--active' : ''}`} onClick={() => handleFormatChange('jpeg')} onKeyDown={(e) => handleKeyDown(e, 'jpeg')}>
                    <div className="format-chip__label">JPEG</div>
                    <div className="format-chip__desc">Maximum compatibility</div>
                </div>
                <div role="button" tabIndex={0} className={`format-chip ${options.format === 'png' ? 'format-chip--active' : ''}`} onClick={() => handleFormatChange('png')} onKeyDown={(e) => handleKeyDown(e, 'png')}>
                    <div className="format-chip__label">PNG</div>
                    <div className="format-chip__desc">Lossless, larger files</div>
                </div>
            </div>

            <div className="options-panel__quality">
                <div className="quality-header">
                    <div className="label-with-tooltip">
                        <label htmlFor="quality-slider">Quality</label>
                        <div className="tooltip-icon">
                            ?
                            <span className="tooltip-text">
                                Lower values reduce file size but introduce visual artifacts.
                                <strong>Recommended: 80%</strong> for the web.
                            </span>
                        </div>
                    </div>
                    <div className="quality-value">{options.quality}</div>
                </div>
                <input
                    id="quality-slider"
                    type="range"
                    className="quality-slider"
                    min="1"
                    max="100"
                    value={options.quality}
                    onChange={(e) => onChange({ ...options, quality: parseInt(e.target.value, 10) })}
                />
                <div className="quality-labels">
                    <span>Smallest file</span>
                    <span>Best quality</span>
                </div>
            </div>

            {expanded && (
                <div className="options-panel__advanced">
                    <div className="option-row">
                        <div className="option-group">
                            <div className="label-with-tooltip">
                                <label htmlFor="maxWidth">Max Width</label>
                                <div className="tooltip-icon">
                                    ?
                                    <span className="tooltip-text">
                                        Downscales wide images to this pixel width. Leave empty to keep original width.
                                        <strong>Recommended: 1920-2560</strong>.
                                    </span>
                                </div>
                            </div>
                            <input id="maxWidth" type="number" className="option-input" placeholder="Auto" value={options.width || ''} onChange={(e) => onChange({ ...options, width: parseInt(e.target.value, 10) || 0 })} />
                        </div>
                        <div className="option-group">
                            <div className="label-with-tooltip">
                                <label htmlFor="maxHeight">Max Height</label>
                                <div className="tooltip-icon">
                                    ?
                                    <span className="tooltip-text">
                                        Downscales tall images to this pixel height. Leave empty to keep original height.
                                    </span>
                                </div>
                            </div>
                            <input id="maxHeight" type="number" className="option-input" placeholder="Auto" value={options.height || ''} onChange={(e) => onChange({ ...options, height: parseInt(e.target.value, 10) || 0 })} />
                        </div>
                    </div>
                    <div className="option-row" style={{ marginTop: '16px', alignItems: 'center' }}>
                        <div className="label-with-tooltip">
                            <label className="option-checkbox">
                                <input type="checkbox" checked={options.stripMetadata} onChange={(e) => onChange({ ...options, stripMetadata: e.target.checked })} />
                                Strip metadata
                            </label>
                            <div className="tooltip-icon">
                                ?
                                <span className="tooltip-text">
                                    Removes EXIF data (camera info, GPS) which saves extra KBs and protects privacy.
                                    <strong>Recommended: ON.</strong>
                                </span>
                            </div>
                        </div>
                        <div className="label-with-tooltip">
                            <label className="option-checkbox">
                                <input type="checkbox" checked={options.lossless} onChange={(e) => onChange({ ...options, lossless: e.target.checked })} />
                                Lossless
                            </label>
                            <div className="tooltip-icon">
                                ?
                                <span className="tooltip-text">
                                    Ensures zero pixel degradation, but file sizes will be significantly larger.
                                    <strong>Recommended: OFF</strong> unless required for strict archiving.
                                </span>
                            </div>
                        </div>
                    </div>
                    <div className="option-group" style={{ marginTop: '16px' }}>
                        <div className="quality-header">
                            <div className="label-with-tooltip">
                                <label htmlFor="effort-slider">Encoding Effort</label>
                                <div className="tooltip-icon">
                                    ?
                                    <span className="tooltip-text" style={{ bottom: '100%', top: 'auto', right: 0, left: 'auto', transform: 'translateX(20px)' }}>
                                        Higher effort compress files slightly smaller, but takes much longer to process.
                                        <strong>Recommended: 4</strong> for a fast balanced workflow.
                                    </span>
                                </div>
                            </div>
                            <div className="quality-value">{options.effort}</div>
                        </div>
                        <input id="effort-slider" type="range" className="quality-slider" min="0" max="9" value={options.effort} onChange={(e) => onChange({ ...options, effort: parseInt(e.target.value, 10) })} />
                        <div className="quality-labels">
                            <span>Faster</span>
                            <span>Smaller</span>
                        </div>
                    </div>
                </div>
            )}

            <button className="process-btn" disabled={fileCount === 0 || processing} onClick={onProcess}>
                {processing ? <div className="spinner"></div> : (
                    <>
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
                        </svg>
                        Optimize {fileCount} file{fileCount !== 1 ? 's' : ''}
                    </>
                )}
            </button>
        </div>
    );
}
