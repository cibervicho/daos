/* Copyright (C) 2016 Intel Corporation
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted for any purpose (including commercial purposes)
 * provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions, and the following disclaimer.
 *
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions, and the following disclaimer in the
 *    documentation and/or materials provided with the distribution.
 *
 * 3. In addition, redistributions of modified forms of the source or binary
 *    code must carry prominent notices stating that the original code was
 *    changed and the date of the change.
 *
 *  4. All publications or advertising materials mentioning features or use of
 *     this software are asked, but not required, to acknowledge that it was
 *     developed by Intel Corporation and credit the contributors.
 *
 * 5. Neither the name of Intel Corporation, nor the name of any Contributor
 *    may be used to endorse or promote products derived from this software
 *    without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

#ifndef __CRT_HASH_H__
#define __CRT_HASH_H__

#include <crt_util/list.h>

#define DHASH_DEBUG	0

struct dhash_table;

typedef struct {
	/**
	 * Optional, generate a key for the record @rlink.
	 *
	 * This function is called before inserting a record w/o key into a
	 * hash table.
	 *
	 * \param rlink	[IN]	The link chain of the record to generate key.
	 * \param args	[IN]	Input arguments for the key generating.
	 */
	void	 (*hop_key_init)(struct dhash_table *htable,
				 crt_list_t *rlink, void *args);
	/**
	 * Optional, return the key of record @rlink to @key_pp, and size of
	 * the key as the returned value.
	 *
	 * \param rlink	[IN]	The link chain of the record being queried.
	 * \param key_pp [OUT]	The returned key.
	 *
	 * \return		size of the key.
	 */
	int	 (*hop_key_get)(struct dhash_table *htable, crt_list_t *rlink,
				void **key_pp);
	/**
	 * Optional, hash @key to a 32-bit value.
	 * DJB2 hash is used when this function is abscent.
	 */
	uint32_t (*hop_key_hash)(struct dhash_table *htable, const void *key,
				 unsigned int ksize);
	/**
	 * Compare @key with the key of the record @rlink
	 * This member function is mandatory.
	 *
	 * \return	true	The key of the record equals to @key.
	 *		false	Not match
	 */
	bool	 (*hop_key_cmp)(struct dhash_table *htable, crt_list_t *rlink,
				const void *key, unsigned int ksize);
	/**
	 * Optional, increase refcount on the record @rlink
	 * If this function is provided, it will be called for successfully
	 * inserted record.
	 *
	 * \param rlink	[IN]	The record being referenced.
	 */
	void	 (*hop_rec_addref)(struct dhash_table *htable,
				    crt_list_t *rlink);
	/**
	 * Optional, release refcount on the record @rlink
	 *
	 * If this function is provided, it is called while deleting an record
	 * from the hash table.
	 *
	 * If hop_free() is also provided, this function can return true when
	 * the refcount reaches zero, in this case, hop_free() will be called.
	 * If the record should not be automatically freed by the hash table
	 * despite of refcount, then this function should never return true.
	 *
	 * \param rlink	[IN]	The rlink being released.
	 *
	 * \return	False	Do nothing
	 *		True	Only if refcount is zero and the hash item
	 *			can be freed. If this function can return
	 *			true, then hop_rec_free() should be defined.
	 */
	bool	 (*hop_rec_decref)(struct dhash_table *htable,
				    crt_list_t *rlink);
	/**
	 * Optional, free the record @rlink
	 * It is called if hop_decref() returns zero.
	 *
	 * \param rlink	[IN]	The record being freed.
	 */
	void	 (*hop_rec_free)(struct dhash_table *htable,
				  crt_list_t *rlink);
} dhash_table_ops_t;

enum dhash_feats {
	/**
	 * By default, the hash table is protected by pthread_mutex.
	 */

	/**
	 * The hash table has no lock, it means the hash table is protected
	 * by external lock, or only accessed by a single thread.
	 */
	DHASH_FT_NOLOCK		= (1 << 0),
	/**
	 * It is a read-mostly hash table, so it is protected by RW lock.
	 *
	 * NB: If caller sets this flag and also provides hop_addref/decref,
	 * then he should guarantee refcount changes are atomic or protected
	 * within hop_addref/decref, because RW lock can't protect refcount.
	 */
	DHASH_FT_RWLOCK		= (1 << 1),
};

/** a hash bucket */
struct dhash_bucket {
	crt_list_t		hb_head;
#if DHASH_DEBUG
	unsigned int		hb_dep;
#endif
};

struct dhash_table {
	/** different type of locks based on ht_feats */
	union {
		pthread_mutex_t		ht_lock;
		pthread_rwlock_t	ht_rwlock;
	};
	/** bits to generate number of buckets */
	unsigned int		 ht_bits;
	/** feature bits */
	unsigned int		 ht_feats;
#if DHASH_DEBUG
	/** maximum search depth ever */
	unsigned int		 ht_dep_max;
	/** maximum number of hash records */
	unsigned int		 ht_nr_max;
	/** total number of hash records */
	unsigned int		 ht_nr;
#endif
	/** private data to pass into customized functions */
	void			*ht_priv;
	/** customized member functions */
	dhash_table_ops_t	*ht_ops;
	/** array of buckets */
	struct dhash_bucket	*ht_buckets;
};


int  dhash_table_create(uint32_t feats, unsigned int bits,
			void *priv, dhash_table_ops_t *hops,
			struct dhash_table **htable_pp);
int  dhash_table_create_inplace(uint32_t feats, unsigned int bits,
				void *priv, dhash_table_ops_t *hops,
				struct dhash_table *htable);
typedef int (*dhash_traverse_cb_t)(crt_list_t *rlink, void *args);
int dhash_table_traverse(struct dhash_table *htable, dhash_traverse_cb_t cb,
			 void *args);
int  dhash_table_destroy(struct dhash_table *htable, bool force);
int  dhash_table_destroy_inplace(struct dhash_table *htable, bool force);
void dhash_table_debug(struct dhash_table *htable);

crt_list_t *dhash_rec_find(struct dhash_table *htable, const void *key,
			    unsigned int ksize);
int  dhash_rec_insert(struct dhash_table *htable, const void *key,
		     unsigned int ksize, crt_list_t *rlink,
		     bool exclusive);
int  dhash_rec_insert_anonym(struct dhash_table *htable, crt_list_t *rlink,
			      void *args);
bool dhash_rec_delete(struct dhash_table *htable, const void *key,
		      unsigned int ksize);
bool dhash_rec_delete_at(struct dhash_table *htable, crt_list_t *rlink);
void dhash_rec_addref(struct dhash_table *htable, crt_list_t *rlink);
void dhash_rec_decref(struct dhash_table *htable, crt_list_t *rlink);
bool dhash_rec_unlinked(crt_list_t *rlink);

#define CRT_HHASH_BITS		16
#define CRT_HTYPE_BITS		3
#define CRT_HTYPE_MASK		((1ULL << CRT_HTYPE_BITS) - 1)

enum {
	CRT_HTYPE_EQ		= 0, /* event queue */
	CRT_HTYPE_POOL		= 1,
	CRT_HTYPE_CO		= 2, /* container */
	CRT_HTYPE_OBJ		= 3, /* object */
	/* More to be added */
};

struct crt_hlink;

struct crt_hlink_ops {
	/** free callback */
	void	(*hop_free)(struct crt_hlink *hlink);
};

struct crt_hlink {
	crt_list_t		 hl_link;
	uint64_t		 hl_key;
	unsigned int		 hl_ref;
	unsigned int		 hl_initialized:1;
	struct crt_hlink_ops	*hl_ops;
};

struct crt_hhash;

int crt_hhash_create(unsigned int bits, struct crt_hhash **hhash);
void crt_hhash_destroy(struct crt_hhash *hh);
void crt_hhash_hlink_init(struct crt_hlink *hlink,
			   struct crt_hlink_ops *ops);
void crt_hhash_link_insert(struct crt_hhash *hhash,
			    struct crt_hlink *hlink, int type);
struct crt_hlink *crt_hhash_link_lookup(struct crt_hhash *hhash,
					  uint64_t key);
void crt_hhash_link_putref(struct crt_hhash *hhash,
			    struct crt_hlink *hlink);
bool crt_hhash_link_delete(struct crt_hhash *hhash,
			    struct crt_hlink *hlink);
bool crt_hhash_link_empty(struct crt_hlink *hlink);
void crt_hhash_link_key(struct crt_hlink *hlink, uint64_t *key);
int crt_hhash_key_type(uint64_t key);

#endif /*__CRT_HASH_H__*/
