/**
 * (C) Copyright 2021 Intel Corporation.
 *
 * SPDX-License-Identifier: BSD-2-Clause-Patent
 */
#define D_LOGFAC DD_FAC(tests)

#include <abt.h>
#include <assert.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <dml.h>
#include "vts_io.h"

#define DEFAULT_NUM_XSTREAMS 3
#define MIN_OPS (1100)
#define OFFSET	(1024 * 64)
#define BUF_SIZE (1024 * 128)

enum {
	NO_DSA_NO_CSUM,
	NO_DSA_CSUM,
	DSA_NO_CSUM,
	DSA_CSUM,
};

ABT_pool *          pools;
struct vos_test_ctx vtx;

static dml_path_t	path = DML_PATH_SW;
static struct daos_csummer	*csummer;
static bool		gen_csum;
static bool		use_dsa;
static daos_epoch_t	epoch = 1;
static daos_unit_oid_t	oid;
static uint64_t		dkey_val = 0;
static char		akey_val = 'a';
static daos_key_t	dkey = {
	.iov_buf = &dkey_val,
	.iov_len = sizeof(dkey_val),
	.iov_buf_len = sizeof(dkey_val),
};
static daos_key_t	akey = {
	.iov_buf = &akey_val,
	.iov_len = sizeof(akey_val),
	.iov_buf_len = sizeof(akey_val),
};

struct io_op {
	daos_epoch_t	epoch;
	daos_iod_t	iod;
	d_sg_list_t	sgl;
	daos_recx_t	recx;
	d_iov_t		sg_iov;
	struct dcs_iod_csums	*iod_csums;
	char		buf[BUF_SIZE];
};

static int	new_io;
static daos_epoch_t highest_io;

void submit_io(void *arg)
{
	int rank;
	int rc;
	struct io_op	*op = arg;

	ABT_xstream_self_rank(&rank);
	rc = vos_obj_update(vtx.tc_co_hdl, oid, op->epoch, 0, 0, &dkey, 1, &op->iod,
			    op->iod_csums, &op->sgl);
	new_io++;
	if (op->epoch > highest_io)
		highest_io = op->epoch;
	D_ASSERT(rc == 0);
}

static dml_job_t *
init_dml_job(void)
{
	dml_job_t	*dml_job_ptr = NULL;
	uint32_t	size = 0;
	dml_status_t	status = dml_get_job_size(path, &size);
	if (status != DML_STATUS_OK)
		return NULL;

	dml_job_ptr = (dml_job_t *)malloc(size);
	status = dml_init_job(path, dml_job_ptr);
	if (status != DML_STATUS_OK) {
		free(dml_job_ptr);
		return NULL;
	}

	return dml_job_ptr;
}

int
handle_write(struct csum_recalc_args *args)
{
	dml_job_t	*job_ptr = init_dml_job();
	dml_status_t	status;
	char		*src, *dest;
	int		rc = 0;
	printf("handle write\n");

	if (job_ptr == NULL)
		return -DER_NOMEM;

	job_ptr->operation = DML_OP_MEM_MOVE;
	job_ptr->source_first_ptr = args->iov->iov_buf;
	src = (char *)job_ptr->source_first_ptr;
	job_ptr->destination_length = job_ptr->source_length = args->iov->iov_len;
	job_ptr->destination_first_ptr = bio_addr2ptr(args->bio_ctx, args->cra_ent_in->ei_addr);
	dest = (char *)job_ptr->destination_first_ptr;

	status = dml_submit_job(job_ptr);
	if (status != DML_STATUS_OK) {
		rc = -DER_INVAL;
		goto out;
	}

	for (;;) {
		status = dml_check_job(job_ptr);
		if (status == DML_STATUS_OK)
			break;

		if (status == DML_STATUS_JOB_CORRUPTED) {
			rc = -DER_INVAL;
			break;
		}

		/** Let another ULT execute */
		ABT_thread_yield();
	}

out:
	printf("src[0] = %c dest[0] = %c memcmp = %d bytes="DF_U64"\n", src[0], dest[0],
	       memcmp(src, dest, args->iov->iov_len), args->iov->iov_len);

	dml_finalize_job(job_ptr);
	ABT_eventual_set(args->csum_eventual, NULL, 0);
	return rc;
}

int
handle_write_csum(struct csum_recalc_args *args)
{
	dml_job_t	*job_ptr = init_dml_job();
	printf("handle write csum\n");

	if (job_ptr == NULL)
		return -DER_NOMEM;

	dml_finalize_job(job_ptr);
	ABT_eventual_set(args->csum_eventual, NULL, 0);

	return 0;
}

int
handle_csum(struct csum_recalc_args *args)
{
	dml_job_t	*job_ptr = init_dml_job();

	printf("handle csum\n");

	if (job_ptr == NULL)
		return -DER_NOMEM;

	dml_finalize_job(job_ptr);
	ABT_eventual_set(args->csum_eventual, NULL, 0);
	return 0;

#if 0
	d_sg_list_t		 sgl;
	struct csum_recalc_args *args = recalc_args;
	struct bio_sglist	*bsgl = args->cra_bsgl;
	struct evt_entry_in	*ent_in = args->cra_ent_in;
	struct csum_recalc	*recalcs = args->cra_recalcs;
	struct daos_csummer	*csummer;
	struct dcs_csum_info	 csum_info = args->cra_ent_in->ei_csum;
	unsigned int		 buf_idx = 0;
	unsigned int		 add_idx = 0;
	unsigned int		 i,  add_offset = 0;
	int			 rc = 0;

	/* need at most prefix + buf + suffix in sgl */
	rc = d_sgl_init(&sgl, 3);
	if (rc) {
		args->cra_rc = rc;
		return;
	}
	daos_csummer_init_with_type(&csummer, csum_info.cs_type,
				    csum_info.cs_chunksize, 0);
	for (i = 0; i < args->cra_seg_cnt; i++) {
		bool		is_valid = false;
		unsigned int	this_buf_nr, this_buf_idx;


		/* Number of records in this input segment, include added
		 * segments.
		 */
		this_buf_nr = (bsgl->bs_iovs[i].bi_data_len +
			       recalcs[i].cr_prefix_len +
			       recalcs[i].cr_suffix_len) / ent_in->ei_inob;
		/* Sets up the SGL for the (verification) checksum calculation.
		 * Returns the offset of the next add-on (prefix/suffix)
		 * segment.
		 */
		add_offset = csum_agg_set_sgl(&sgl, bsgl, recalcs,
					      args->cra_buf, args->cra_buf_len,
					      args->cra_seg_cnt,
					      args->cra_seg_size, i, add_offset,
					      &buf_idx, &add_idx);
		D_ASSERT(recalcs[i].cr_log_ext.ex_hi -
			 recalcs[i].cr_log_ext.ex_lo + 1 ==
			 bsgl->bs_iovs[i].bi_data_len / ent_in->ei_inob);

		/* Determines number of checksum entries, and start index, for
		 * calculating verification checksum,
		 */
		this_buf_idx = calc_csum_params(&csum_info, &recalcs[i],
						recalcs[i].cr_prefix_len,
						recalcs[i].cr_suffix_len,
						ent_in->ei_inob);

		/* Ensure buffer is zero-ed. */
		memset(csum_info.cs_csum, 0, csum_info.cs_buf_len);

		/* Calculates the checksums for the input segment. */
		rc = daos_csummer_calc_one(csummer, &sgl, &csum_info,
					   ent_in->ei_inob, this_buf_nr,
					   this_buf_idx);
		if (rc)
			goto out;

		/* Verifies that calculated checksums match prior (input)
		 * checksums, for the appropriate range.
		 */
		is_valid = csum_agg_verify(&recalcs[i], &csum_info,
					   ent_in->ei_inob,
					   recalcs[i].cr_prefix_len);
		if (!is_valid) {
			rc = -DER_CSUM;
			goto out;
		}
	}
out:
	/* Eventual set okay, even with no offload (unit test). */
	D_FREE(sgl.sg_iovs);
	args->cra_rc = rc;
	return 0;
#endif
}

void
agg_csum_recalc(void *recalc_args)
{
	struct csum_recalc_args *args = recalc_args;
	int rc;

	if (!use_dsa) {
		printf("recalc called\n");
		ds_csum_agg_recalc(recalc_args);
		return;
	}

	if (args->is_write) {
		if (gen_csum)
			rc = handle_write_csum(args);
		else
			rc = handle_write(args);
	} else {
		rc = handle_csum(args);
	}

	args->cra_rc = rc;
}

void
csum_recalc(void *args)
{
	struct csum_recalc_args *cs_args = args;
	ABT_pool	target_pool = pools[2];
	ABT_thread	thread;
	int rc;

	if (!use_dsa && cs_args->is_write) {
		printf("bio_write called\n");
		rc = bio_write(cs_args->bio_ctx, cs_args->cra_ent_in->ei_addr, cs_args->iov);
		cs_args->cra_rc = rc;
		return;
	}

	ABT_eventual_create(0, &cs_args->csum_eventual);
	ABT_thread_create(target_pool, agg_csum_recalc, args, ABT_THREAD_ATTR_NULL, &thread);
	ABT_eventual_wait(cs_args->csum_eventual, NULL);
	ABT_eventual_free(&cs_args->csum_eventual);
}

void agg_thread(void *arg)
{
	int *done = arg;
	daos_epoch_range_t	epr = {0};
	int rc;

	while (!done || new_io != 0) {
		if (epr.epr_hi == highest_io && new_io == 0) {
			ABT_thread_yield();
			continue;
		}

		new_io = 0;
		epr.epr_lo = 0;
		epr.epr_hi = highest_io;
		rc = vos_aggregate(vtx.tc_co_hdl, &epr, csum_recalc, NULL, NULL, true);
		D_ASSERT(rc == 0);
	}
}

struct io_op *
allocate_ops(int op_count, bool csum)
{
	int i;
	uint64_t offset = OFFSET;
	struct io_op *ops;

	ops = malloc(sizeof(*ops) * op_count);

	for (i = 0; i < op_count; i++) {
		ops[i].epoch = epoch++;
		ops[i].iod.iod_name = akey;
		ops[i].iod.iod_type = DAOS_IOD_ARRAY;
		ops[i].iod.iod_size = 1;
		ops[i].iod.iod_recxs = &ops[i].recx;
		ops[i].iod.iod_nr = 1;
		ops[i].sgl.sg_nr_out = 0;
		ops[i].sgl.sg_nr = 1;
		ops[i].sgl.sg_iovs = &ops[i].sg_iov;
		memset(ops[i].buf, (i % 26) + 'A', sizeof(ops[i].buf));
		d_iov_set(&ops[i].sg_iov, ops[i].buf, sizeof(ops[i].buf));
		ops[i].recx.rx_nr = BUF_SIZE;
		ops[i].recx.rx_idx = offset;
		offset += BUF_SIZE;
		if (csum) {
			daos_csummer_calc_iods(csummer, &ops[i].sgl, &ops[i].iod, NULL, 1, false,
					       NULL, 0, &ops[i].iod_csums);
		}
	}

	return ops;
}

void run_bench(int num_init, int num_ops)
{
	int         i;
	ABT_pool    target_pool = pools[1];
	ABT_thread *children    = malloc(sizeof(*children) * (num_ops + 1));
	struct io_op	*args;
	int		rc;
	int		done = 0;

	oid = dts_unit_oid_gen(0, DAOS_OF_DKEY_UINT64, 0);

	if (gen_csum) {
		rc = daos_csummer_init_with_type(&csummer, HASH_TYPE_CRC32, 1 << 12, 0);
		D_ASSERT(rc == 0);
	}

	args = allocate_ops(num_ops, gen_csum);

	for (i = 0; i < num_init; i++) {
		ABT_thread_create(target_pool, submit_io, &args[i], ABT_THREAD_ATTR_NULL,
				  &children[i]);
	}

	for (i = 0; i < num_init; i++) {
		ABT_thread_free(&children[i]);
	}

	ABT_thread_create(target_pool, agg_thread, &done, ABT_THREAD_ATTR_NULL,
			  &children[num_ops]);
	for (i = num_init; i < num_ops; i++) {
		ABT_thread_create(target_pool, submit_io, &args[i], ABT_THREAD_ATTR_NULL,
				  &children[i]);
	}

	for (i = num_init; i < num_ops + 1; i++) {
		if (i == num_ops)
			done = 1; /** Nothing left but to wait for agg thread to finish */
		ABT_thread_free(&children[i]);
	}

	if (gen_csum) {
		daos_csummer_destroy(&csummer);
	}
}

void
print_usage(const char *name)
{
	printf("Usage: %s [opts]\n", name);
	printf("\t-h            Print help and exit\n");
	printf("\t-n count      Set number of operations to perform\n");
	printf("\t-t d|D|n|N    Aggregation with(d) or without DSA(d), capitalize for csum, default is 'n'\n");
	printf("\t-p s|h	s is software DSA, h is hardware DSA\n");
}

int main(int argc, char **argv)
{
	int i, rc, num_ops = 0, num_init;
	/* Read arguments. */
	int num_xstreams = DEFAULT_NUM_XSTREAMS;
	while (1) {
		int opt = getopt(argc, argv, "hn:t:p:");
		if (opt == -1)
			break;
		switch (opt) {
		case 'p':
			switch (optarg[0]) {
			case 's':
				path = DML_PATH_SW;
				break;
			case 'h':
				path = DML_PATH_HW;
				break;
			default:
				print_usage(argv[0]);
				return -1;
			}
			break;
		case 'o':
			num_ops = atoi(optarg);
			break;
		case 't':
			switch (optarg[0]) {
			case 'D':
				gen_csum = true;
				/* fallthrough */
			case 'd':
				use_dsa = true;
				break;
			case 'N':
				gen_csum = true;
				/* fallthrough */
			case 'n':
				break;
			default:
				print_usage(argv[0]);
				return -1;
			}
			break;
		case 'h':
		default:
			print_usage(argv[0]);
			return -1;
		}
	}

	if (num_ops < MIN_OPS)
		num_ops = MIN_OPS;

	num_init = num_ops / 11;

	rc = daos_debug_init(DAOS_LOG_DEFAULT);
	if (rc) {
		print_error("Error initializing debug system\n");
		return rc;
	}

	rc = vos_self_init("/mnt/daos");
	if (rc) {
		print_error("Error initializing VOS instance\n");
		goto exit_0;
	}
	vts_ctx_init(&vtx, VPOOL_10G);

	/* Allocate memory. */
	ABT_xstream *xstreams = (ABT_xstream *)malloc(sizeof(ABT_xstream) * num_xstreams);
	pools                 = (ABT_pool *)malloc(sizeof(ABT_pool) * num_xstreams);
	ABT_sched *scheds     = (ABT_sched *)malloc(sizeof(ABT_sched) * num_xstreams);

	/* Initialize Argobots. */
	ABT_init(argc, argv);

	/* Create pools. */
	for (i = 0; i < num_xstreams; i++) {
		ABT_pool_create_basic(ABT_POOL_FIFO, ABT_POOL_ACCESS_MPMC, ABT_TRUE, &pools[i]);
	}

	/* Create schedulers. */
	for (i = 0; i < num_xstreams; i++) {
		ABT_sched_create_basic(ABT_SCHED_DEFAULT, 1, &pools[i], ABT_SCHED_CONFIG_NULL,
				       &scheds[i]);
	}

	/* Set up a primary execution stream. */
	ABT_xstream_self(&xstreams[0]);
	ABT_xstream_set_main_sched(xstreams[0], scheds[0]);

	/* Create secondary execution streams. */
	for (i = 1; i < num_xstreams; i++) {
		ABT_xstream_create(scheds[i], &xstreams[i]);
	}

	run_bench(num_init, num_ops);

	/* Join secondary execution streams. */
	for (i = 1; i < num_xstreams; i++) {
		ABT_xstream_join(xstreams[i]);
		ABT_xstream_free(&xstreams[i]);
	}

	/* Finalize Argobots. */
	ABT_finalize();

	/* Free allocated memory. */
	free(xstreams);
	free(pools);
	free(scheds);

	vts_ctx_fini(&vtx);
	vos_self_fini();
exit_0:
	daos_debug_fini();

	return 0;
}
