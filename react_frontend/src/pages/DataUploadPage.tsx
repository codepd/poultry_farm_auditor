import React, { useState, useRef, useEffect, useCallback } from 'react';
import { adminAPI, DocumentUploadRecord } from '../services/api';
import { useNavigate } from 'react-router-dom';
import './DataUploadPage.css';

const DataUploadPage: React.FC = () => {
  const [dragActive, setDragActive] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [parsing, setParsing] = useState(false);
  const [asyncStage, setAsyncStage] = useState<string>('');
  const [committing, setCommitting] = useState(false);
  const [error, setError] = useState('');
  const [parsedData, setParsedData] = useState<any | null>(null);
  const [currentUploadId, setCurrentUploadId] = useState<number | null>(null);
  const [committedSuccess, setCommittedSuccess] = useState(false);
  const [archivedFeedCard, setArchivedFeedCard] = useState<any | null>(null);
  const [activeTab, setActiveTab] = useState<'feeds' | 'eggs' | 'discounts' | 'payments'>('feeds');

  // Audit history state
  const [uploadHistory, setUploadHistory] = useState<DocumentUploadRecord[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [typeFilter, setTypeFilter] = useState<string>('ALL');
  const [yearFilter, setYearFilter] = useState<string>('ALL');

  const fileInputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  const fetchUploadHistory = useCallback(async () => {
    setLoadingHistory(true);
    try {
      const records = await adminAPI.listUploads({
        file_type: typeFilter !== 'ALL' ? typeFilter : undefined,
        year: yearFilter !== 'ALL' ? yearFilter : undefined,
      });
      setUploadHistory(records);
    } catch (err) {
      // Non-blocking if table is empty or offline
    } finally {
      setLoadingHistory(false);
    }
  }, [typeFilter, yearFilter]);

  useEffect(() => {
    fetchUploadHistory();
  }, [fetchUploadHistory]);

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFile(e.dataTransfer.files[0]);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault();
    if (e.target.files && e.target.files[0]) {
      handleFile(e.target.files[0]);
    }
  };

  const handleFile = (file: File) => {
    const ext = file.name.substring(file.name.lastIndexOf('.')).toLowerCase();
    const allowed = ['.pdf', '.xlsx', '.xls', '.jpg', '.jpeg', '.png', '.webp'];
    if (!allowed.includes(ext)) {
      setError('Please select a valid Ledger PDF (.pdf), Excel (.xlsx, .xls), or Feed Price Card (.jpg, .jpeg, .png)');
      return;
    }
    setSelectedFile(file);
    setError('');
    setParsedData(null);
    setCurrentUploadId(null);
    setCommittedSuccess(false);
    setArchivedFeedCard(null);
  };

  const pollUploadStatus = async (uploadId: number): Promise<DocumentUploadRecord> => {
    for (let attempt = 0; attempt < 45; attempt++) {
      await new Promise((resolve) => setTimeout(resolve, 1200));
      const statusRec = await adminAPI.getUploadStatus(uploadId);
      if (statusRec.upload_status === 'PARSED' || statusRec.upload_status === 'COMMITTED' || statusRec.upload_status === 'ARCHIVED') {
        return statusRec;
      }
      if (statusRec.upload_status === 'ERROR') {
        throw new Error(statusRec.error_message || 'Background parser reported an error');
      }
    }
    throw new Error('Statement parsing timed out. Check the S3 Audit Vault below for status updates.');
  };

  const handleParsePreview = async () => {
    if (!selectedFile) return;
    setParsing(true);
    setError('');
    setAsyncStage('Uploading document to AWS S3 Audit Vault...');
    try {
      const initialResp = await adminAPI.uploadLedger(selectedFile, false, { async: true });
      if (initialResp.upload_status === 'ARCHIVED' || initialResp.file_type === 'FEED_PRICE_IMAGE') {
        setArchivedFeedCard(initialResp);
        fetchUploadHistory();
        return;
      }

      if (initialResp.async && initialResp.upload_id) {
        setCurrentUploadId(initialResp.upload_id);
        setAsyncStage('Stored in S3 • Running async ledger extraction & balance check...');
        const completedRec = await pollUploadStatus(initialResp.upload_id);
        const merged = {
          ...(completedRec.parse_result || {}),
          upload_id: completedRec.id,
          s3_bucket: completedRec.s3_bucket,
          s3_key: completedRec.s3_key,
          sha256_hash: completedRec.sha256_hash,
          download_url: completedRec.download_url,
        };
        setParsedData(merged);
      } else {
        setCurrentUploadId(initialResp.upload_id || null);
        setParsedData(initialResp);
      }
      fetchUploadHistory();
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Failed to upload or parse document');
    } finally {
      setParsing(false);
      setAsyncStage('');
    }
  };

  const handleCommit = async () => {
    setCommitting(true);
    setError('');
    try {
      if (currentUploadId) {
        await adminAPI.commitUpload(currentUploadId);
      } else if (selectedFile) {
        await adminAPI.uploadLedger(selectedFile, true);
      }
      setCommittedSuccess(true);
      fetchUploadHistory();
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Failed to commit ledger to database');
    } finally {
      setCommitting(false);
    }
  };

  const handleReviewHistoricalRecord = (rec: DocumentUploadRecord) => {
    if (!rec.parse_result) return;
    setError('');
    setCommittedSuccess(false);
    setArchivedFeedCard(null);
    setCurrentUploadId(rec.id);
    setParsedData({
      ...rec.parse_result,
      upload_id: rec.id,
      s3_bucket: rec.s3_bucket,
      s3_key: rec.s3_key,
      sha256_hash: rec.sha256_hash,
      download_url: rec.download_url,
    });
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleReset = () => {
    setSelectedFile(null);
    setParsedData(null);
    setCurrentUploadId(null);
    setCommittedSuccess(false);
    setArchivedFeedCard(null);
    setError('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const formatCurrency = (val: number | undefined | null) => {
    if (val === undefined || val === null) return '₹0.00';
    return `₹${val.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;
  };

  const formatNumber = (val: number | undefined | null) => {
    if (val === undefined || val === null) return '0';
    return val.toLocaleString('en-IN', { maximumFractionDigits: 2 });
  };

  const formatBytes = (bytes?: number) => {
    if (!bytes) return '-';
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
    return `${(bytes / 1024).toFixed(1)} KB`;
  };

  const formatPeriod = (year?: number, month?: number) => {
    if (!year) return '-';
    if (!month) return String(year);
    const d = new Date(year, month - 1, 1);
    return `${d.toLocaleString('en-US', { month: 'short' })} ${year}`;
  };

  const summary = parsedData?.summary;

  return (
    <div className="data-upload-page">
      <div className="upload-header">
        <div>
          <h2>Monthly Data Sheet & S3 Audit Portal</h2>
          <span className="upload-subtitle">
            Upload monthly ledger PDFs, parsed Excel sheets, or feed price rate cards. All files are archived in AWS S3 with SHA-256 verification and parsed asynchronously.
          </span>
        </div>
      </div>

      {error && <div className="upload-error">{error}</div>}

      {archivedFeedCard && (
        <div className="commit-success-card">
          <div className="success-icon">☁️</div>
          <h3>Feed Price Sheet Archived to AWS S3!</h3>
          <p>
            <strong>{archivedFeedCard.file_name}</strong> has been stored in <code>s3://{archivedFeedCard.s3_bucket}/{archivedFeedCard.s3_key}</code> for permanent audit reference.
          </p>
          <div className="success-actions">
            {archivedFeedCard.download_url && (
              <a
                href={archivedFeedCard.download_url}
                target="_blank"
                rel="noopener noreferrer"
                className="view-btn"
                style={{ textDecoration: 'none' }}
              >
                View S3 Object
              </a>
            )}
            <button className="another-btn" onClick={handleReset}>
              Upload Another Document
            </button>
          </div>
        </div>
      )}

      {committedSuccess && (
        <div className="commit-success-card">
          <div className="success-icon">✅</div>
          <h3>Statement Validated, Archived in S3 & Imported!</h3>
          <p>
            {parsedData?.pdf_filename || selectedFile?.name} for <strong>{summary?.ledger_period || `${parsedData?.month}/${parsedData?.year}`}</strong> is archived at <code>s3://{parsedData?.s3_bucket}/{parsedData?.s3_key}</code> and committed to your farm database.
          </p>
          <div className="success-actions">
            <button
              className="view-btn"
              onClick={() => navigate('/')}
            >
              View Month in Dashboard
            </button>
            <button
              className="another-btn"
              onClick={handleReset}
            >
              Upload Another Statement
            </button>
          </div>
        </div>
      )}

      {!committedSuccess && !archivedFeedCard && !parsedData && (
        <div className="upload-box-wrapper">
          <form
            className={`upload-dropzone ${dragActive ? 'drag-active' : ''}`}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
            onSubmit={(e) => e.preventDefault()}
          >
            <input
              ref={fileInputRef}
              type="file"
              id="file-upload-input"
              accept=".pdf,.xlsx,.xls,.jpg,.jpeg,.png,.webp"
              onChange={handleChange}
              style={{ display: 'none' }}
            />
            <div className="dropzone-content">
              <span className="drop-icon">☁️</span>
              <h3>Drag & drop your Ledger PDF, Excel, or Feed Price Sheet</h3>
              <p>Supports Sakthi & Everest Feeds Statements (.pdf, .xlsx) and Feed Rate Card Images (.jpg, .png)</p>
              <button
                type="button"
                className="select-file-btn"
                onClick={() => fileInputRef.current?.click()}
              >
                Browse File
              </button>
            </div>
          </form>

          {selectedFile && (
            <div className="selected-file-panel">
              <div className="file-info">
                <span className="file-name">{selectedFile.name}</span>
                <span className="file-size">({formatBytes(selectedFile.size)})</span>
              </div>
              <div className="file-actions">
                <button
                  type="button"
                  className="cancel-btn"
                  onClick={handleReset}
                  disabled={parsing}
                >
                  Clear
                </button>
                <button
                  type="button"
                  className="parse-btn"
                  onClick={handleParsePreview}
                  disabled={parsing}
                >
                  {parsing ? (asyncStage || 'Uploading & Parsing...') : 'Upload to S3 & Parse Statement'}
                </button>
              </div>
            </div>
          )}

          <div className="features-info-grid">
            <div className="feature-item">
              <span className="f-icon">☁️</span>
              <h4>AWS S3 Audit Vault</h4>
              <p>Every uploaded ledger and feed price card is versioned in S3 with SHA-256 checksums for full audit traceability.</p>
            </div>
            <div className="feature-item">
              <span className="f-icon">⚡</span>
              <h4>Async Extraction</h4>
              <p>Parses eggs sold, layer vs chick/grower feeds, medicines, credit notes, and cash receipts in the background.</p>
            </div>
            <div className="feature-item">
              <span className="f-icon">🔒</span>
              <h4>Pre-Commit Verification</h4>
              <p>Inspect extracted line items and opening/closing balance math before committing to production records.</p>
            </div>
          </div>
        </div>
      )}

      {/* Pre-Commit Review & Verification View */}
      {!committedSuccess && !archivedFeedCard && parsedData && summary && (
        <div className="parsed-review-card">
          <div className="review-header">
            <div>
              <h3>Statement Validation Preview</h3>
              <span className="review-meta">
                Provider: <strong>{summary.feed_provider || 'Feed Provider'}</strong> | Period: <strong>{summary.ledger_period || `${parsedData.month}/${parsedData.year}`}</strong>
                {parsedData.s3_key && (
                  <> | S3 Audit Key: <code>{parsedData.s3_key}</code></>
                )}
              </span>
            </div>
            <div className="validation-pill-box">
              {parsedData.download_url && (
                <a
                  href={parsedData.download_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="s3-link-pill"
                >
                  ⬇ S3 Original
                </a>
              )}
              {summary.validation_difference === 0 || summary.validation_difference < 1 ? (
                <span className="badge-pass">✓ Statement Math Balanced</span>
              ) : (
                <span className="badge-warn">Diff: {formatCurrency(summary.validation_difference)}</span>
              )}
            </div>
          </div>

          {/* Quick Metrics Bar */}
          <div className="review-metrics-grid">
            <div className="review-metric">
              <span className="r-label">Total Egg Revenue</span>
              <span className="r-val">{formatCurrency(summary.total_eggs)}</span>
            </div>
            <div className="review-metric">
              <span className="r-label">Total Feed Invoiced</span>
              <span className="r-val">{formatCurrency(summary.total_feeds)}</span>
            </div>
            <div className="review-metric">
              <span className="r-label">Medicines & Supplements</span>
              <span className="r-val">{formatCurrency(summary.total_medicines)}</span>
            </div>
            <div className="review-metric">
              <span className="r-label">Discounts / Credit Notes</span>
              <span className="r-val positive">+{formatCurrency(summary.total_discounts)}</span>
            </div>
            <div className="review-metric highlight">
              <span className="r-label">Statement Net Profit</span>
              <span className="r-val">{formatCurrency(summary.net_profit)}</span>
            </div>
            <div className="review-metric">
              <span className="r-label">Payments Received</span>
              <span className="r-val">{formatCurrency(summary.total_payments)}</span>
            </div>
          </div>

          {/* Balance Check */}
          <div className="balance-check-banner">
            <span>Opening Balance: <strong>{formatCurrency(summary.opening_balance)}</strong></span>
            <span>+ Net Statement: <strong>{formatCurrency(summary.net_profit)}</strong></span>
            <span>- Payments: <strong>{formatCurrency(summary.total_payments)}</strong></span>
            <span>= Closing Balance: <strong>{formatCurrency(summary.closing_balance)}</strong></span>
          </div>

          {/* Tabbed Line Items Table */}
          <div className="line-items-section">
            <div className="tab-buttons">
              <button
                className={activeTab === 'feeds' ? 'active' : ''}
                onClick={() => setActiveTab('feeds')}
              >
                Feeds ({parsedData.feeds?.length || 0})
              </button>
              <button
                className={activeTab === 'eggs' ? 'active' : ''}
                onClick={() => setActiveTab('eggs')}
              >
                Eggs ({parsedData.eggs?.length || 0})
              </button>
              <button
                className={activeTab === 'discounts' ? 'active' : ''}
                onClick={() => setActiveTab('discounts')}
              >
                Discounts & Credit Notes ({parsedData.discounts?.length || 0})
              </button>
              <button
                className={activeTab === 'payments' ? 'active' : ''}
                onClick={() => setActiveTab('payments')}
              >
                Payments ({parsedData.payments?.length || 0})
              </button>
            </div>

            <div className="table-wrapper">
              {activeTab === 'feeds' && (
                <table className="review-table">
                  <thead>
                    <tr>
                      <th>Feed Item</th>
                      <th>Quantity</th>
                      <th>Unit</th>
                      <th>Total Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {parsedData.feeds?.map((item: any, i: number) => (
                      <tr key={i}>
                        <td className="item-name-cell">
                          {item.item_name}
                          {item.item_name.toUpperCase().includes('GM') || item.item_name.toUpperCase().includes('GROWER') ? (
                            <span className="tag-grower">Grower</span>
                          ) : item.item_name.toUpperCase().includes('CRUMBLE') || item.item_name.toUpperCase().includes('CHICK') ? (
                            <span className="tag-chick">Chick</span>
                          ) : (
                            <span className="tag-layer">Layer</span>
                          )}
                        </td>
                        <td>{formatNumber(item.total_qty)}</td>
                        <td>{item.unit || 'KGS'}</td>
                        <td className="amount-cell">{formatCurrency(item.total_amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {activeTab === 'eggs' && (
                <table className="review-table">
                  <thead>
                    <tr>
                      <th>Egg Type</th>
                      <th>Quantity (Nos)</th>
                      <th>Total Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {parsedData.eggs?.map((item: any, i: number) => (
                      <tr key={i}>
                        <td>{item.item_name}</td>
                        <td>{formatNumber(item.total_qty)} Nos</td>
                        <td className="amount-cell">{formatCurrency(item.total_amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {activeTab === 'discounts' && (
                <table className="review-table">
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Description</th>
                      <th>Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {parsedData.discounts?.map((item: any, i: number) => (
                      <tr key={i}>
                        <td>{item.date || '-'}</td>
                        <td>{item.raw || 'Credit Note'}</td>
                        <td className="amount-cell positive">+{formatCurrency(item.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {activeTab === 'payments' && (
                <table className="review-table">
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Description</th>
                      <th>Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {parsedData.payments?.map((item: any, i: number) => (
                      <tr key={i}>
                        <td>{item.date || '-'}</td>
                        <td>{item.raw || 'Payment'}</td>
                        <td className="amount-cell">{formatCurrency(item.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Action Footer */}
          <div className="review-actions-bar">
            <button
              type="button"
              className="back-btn"
              onClick={handleReset}
              disabled={committing}
            >
              Cancel / Back to Upload
            </button>
            <button
              type="button"
              className="confirm-commit-btn"
              onClick={handleCommit}
              disabled={committing}
            >
              {committing ? 'Committing to Database...' : '✓ Approve & Save to Farm Records'}
            </button>
          </div>
        </div>
      )}

      {/* S3 Document Audit Vault & Upload History */}
      <div className="audit-vault-section">
        <div className="audit-vault-header">
          <div>
            <h3>☁️ AWS S3 Document Audit Vault</h3>
            <span className="audit-vault-subtitle">
              Immutable archive of monthly ledgers, feed price sheets, and parsed Excel spreadsheets stored in S3.
            </span>
          </div>
          <div className="audit-vault-filters">
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="vault-filter-select"
            >
              <option value="ALL">All Document Types</option>
              <option value="LEDGER_PDF">Ledger PDFs</option>
              <option value="FEED_PRICE_IMAGE">Feed Price History</option>
              <option value="PARSED_EXCEL">Parsed Excels</option>
            </select>
            <select
              value={yearFilter}
              onChange={(e) => setYearFilter(e.target.value)}
              className="vault-filter-select"
            >
              <option value="ALL">All Years</option>
              <option value="2026">2026</option>
              <option value="2025">2025</option>
            </select>
            <button
              type="button"
              className="vault-refresh-btn"
              onClick={fetchUploadHistory}
              disabled={loadingHistory}
            >
              {loadingHistory ? 'Refreshing...' : '↻ Refresh'}
            </button>
          </div>
        </div>

        <div className="table-wrapper">
          <table className="review-table audit-vault-table">
            <thead>
              <tr>
                <th>Document</th>
                <th>Category</th>
                <th>Period</th>
                <th>S3 Object Key</th>
                <th>Size / SHA-256</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {uploadHistory.length === 0 ? (
                <tr>
                  <td colSpan={7} className="empty-vault-cell">
                    {loadingHistory
                      ? 'Loading S3 audit records...'
                      : 'No documents recorded yet. Upload a ledger or run the one-time S3 migration script.'}
                  </td>
                </tr>
              ) : (
                uploadHistory.map((rec) => (
                  <tr key={rec.id}>
                    <td className="item-name-cell">
                      <strong>{rec.file_name}</strong>
                      <div className="vault-source-meta">
                        Source: {rec.upload_source} • {new Date(rec.created_at).toLocaleDateString('en-IN')}
                      </div>
                    </td>
                    <td>
                      <span className={`doc-type-badge type-${rec.file_type.toLowerCase()}`}>
                        {rec.file_type === 'LEDGER_PDF'
                          ? '📄 Ledger PDF'
                          : rec.file_type === 'FEED_PRICE_IMAGE'
                          ? '🏷️ Feed Price Card'
                          : '📊 Parsed Excel'}
                      </span>
                    </td>
                    <td>{formatPeriod(rec.document_year, rec.document_month)}</td>
                    <td className="s3-key-cell">
                      <code>s3://{rec.s3_bucket}/{rec.s3_key}</code>
                    </td>
                    <td>
                      <div>{formatBytes(rec.file_size)}</div>
                      {rec.sha256_hash && (
                        <code className="sha-badge" title={`SHA-256: ${rec.sha256_hash}`}>
                          #{rec.sha256_hash.substring(0, 10)}
                        </code>
                      )}
                    </td>
                    <td>
                      <span className={`upload-status-pill status-${rec.upload_status.toLowerCase()}`}>
                        {rec.upload_status}
                      </span>
                    </td>
                    <td className="vault-actions-cell">
                      {rec.parse_result && (
                        <button
                          type="button"
                          className="vault-action-btn review"
                          onClick={() => handleReviewHistoricalRecord(rec)}
                        >
                          Inspect
                        </button>
                      )}
                      {rec.download_url && (
                        <a
                          href={rec.download_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="vault-action-btn download"
                        >
                          S3 File ↗
                        </a>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default DataUploadPage;
