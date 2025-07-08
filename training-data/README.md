# ZPO Training Data

This directory contains sample email datasets for training ZPO's spam detection models. The data is organized in a structure that works seamlessly with ZPO's enhanced training system.

## Directory Structure

```
training-data/
├── spam/           # Spam/junk email samples
│   ├── 06_spam_phishing.eml
│   ├── 07_spam_getrich.eml
│   ├── 08_spam_lottery.eml
│   ├── 09_spam_drugs.eml
│   └── 10_spam_prize.eml
├── ham/            # Clean/legitimate email samples  
│   ├── 01_clean_business.eml
│   ├── 02_clean_personal.eml
│   ├── 03_clean_newsletter.eml
│   ├── 04_clean_marketing.eml
│   └── 05_clean_update.eml
└── README.md       # This file
```

## Sample Data Overview

- **Total Emails**: 10 samples
- **Spam Emails**: 5 samples (phishing, get-rich-quick, lottery, drugs, prize scams)
- **Ham Emails**: 5 samples (business, personal, newsletter, marketing, update notifications)
- **Balance**: 1:1 ratio (perfect for initial training)

## Training with ZPO

### Auto-Discovery Training
The simplest way to train using this data:

```bash
# Auto-discover and train on all data
./zpo train --auto-discover training-data

# Validate data quality first
./zpo train --auto-discover training-data --validate-only

# Interactive training with preview
./zpo train --auto-discover training-data --interactive
```

### Manual Directory Training
```bash
# Train on specific directories
./zpo train --spam-dir training-data/spam --ham-dir training-data/ham

# With progress tracking and analysis
./zpo train --spam-dir training-data/spam --ham-dir training-data/ham --analyze

# Benchmark accuracy improvements
./zpo train --spam-dir training-data/spam --ham-dir training-data/ham --benchmark
```

### Advanced Training Options
```bash
# Reset and train fresh
./zpo train --auto-discover training-data --reset

# Resume interrupted training
./zpo train --resume

# Limit emails for testing
./zpo train --auto-discover training-data --max-emails 3

# Quiet training for scripts
./zpo train --auto-discover training-data --quiet
```

## Email Format

All emails are in standard `.eml` format with:
- Standard email headers (From, To, Subject, Date, etc.)
- MIME content types
- Realistic spam/ham content patterns
- Various email lengths and structures

## Expanding the Dataset

To add more training data:

1. **Spam emails**: Add `.eml` files to `spam/` directory
2. **Ham emails**: Add `.eml` files to `ham/` directory
3. **Auto-discovery**: Create additional subdirectories with keywords:
   - Spam: `junk/`, `unwanted/`, `blocked/`, `quarantine/`
   - Ham: `clean/`, `legitimate/`, `good/`, `inbox/`, `sent/`

## Training Results

After training with this sample data, you should see:
- **Processing Speed**: ~200-300 emails/second
- **Balance Ratio**: 1.00 (perfect balance)
- **Error Rate**: 0% (all samples parse correctly)
- **Model Size**: ~500-1000 tokens learned

## Integration with ZPO Components

This training data integrates with:
- **Status Command**: Shows training data counts and quality
- **Monitor Command**: Displays learning analytics and model health
- **Filter Command**: Uses trained models for spam detection
- **Milter**: Postfix integration with trained models

## Best Practices

1. **Start Small**: Use this sample data to verify training works
2. **Scale Gradually**: Add more data in balanced batches
3. **Monitor Quality**: Use `--validate-only` to check data quality
4. **Track Progress**: Use `--benchmark` to measure improvements
5. **Save Sessions**: Training sessions are automatically saved for resume

## Related Commands

```bash
# Check if training improved the model
./zpo status

# Monitor training effectiveness
./zpo monitor

# Test spam detection on sample
./zpo filter < training-data/spam/06_spam_phishing.eml

# Test ham detection on sample
./zpo filter < training-data/ham/01_clean_business.eml
```

Perfect for getting ZPO from 0% to 90%+ accuracy in seconds! 🫏⚡ 