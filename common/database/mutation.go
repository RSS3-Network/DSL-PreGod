// Code generated by entc, DO NOT EDIT.

package database

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/naturalselectionlabs/pregod/common/database/predicate"
	"github.com/naturalselectionlabs/pregod/common/database/transaction"
	"github.com/naturalselectionlabs/pregod/common/database/transfer"

	"entgo.io/ent"
)

const (
	// Operation types.
	OpCreate    = ent.OpCreate
	OpDelete    = ent.OpDelete
	OpDeleteOne = ent.OpDeleteOne
	OpUpdate    = ent.OpUpdate
	OpUpdateOne = ent.OpUpdateOne

	// Node types.
	TypeTransaction = "Transaction"
	TypeTransfer    = "Transfer"
)

// TransactionMutation represents an operation that mutates the Transaction nodes in the graph.
type TransactionMutation struct {
	config
	op            Op
	typ           string
	id            *int
	hash          *string
	clearedFields map[string]struct{}
	done          bool
	oldValue      func(context.Context) (*Transaction, error)
	predicates    []predicate.Transaction
}

var _ ent.Mutation = (*TransactionMutation)(nil)

// transactionOption allows management of the mutation configuration using functional options.
type transactionOption func(*TransactionMutation)

// newTransactionMutation creates new mutation for the Transaction entity.
func newTransactionMutation(c config, op Op, opts ...transactionOption) *TransactionMutation {
	m := &TransactionMutation{
		config:        c,
		op:            op,
		typ:           TypeTransaction,
		clearedFields: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// withTransactionID sets the ID field of the mutation.
func withTransactionID(id int) transactionOption {
	return func(m *TransactionMutation) {
		var (
			err   error
			once  sync.Once
			value *Transaction
		)
		m.oldValue = func(ctx context.Context) (*Transaction, error) {
			once.Do(func() {
				if m.done {
					err = errors.New("querying old values post mutation is not allowed")
				} else {
					value, err = m.Client().Transaction.Get(ctx, id)
				}
			})
			return value, err
		}
		m.id = &id
	}
}

// withTransaction sets the old Transaction of the mutation.
func withTransaction(node *Transaction) transactionOption {
	return func(m *TransactionMutation) {
		m.oldValue = func(context.Context) (*Transaction, error) {
			return node, nil
		}
		m.id = &node.ID
	}
}

// Client returns a new `ent.Client` from the mutation. If the mutation was
// executed in a transaction (ent.Tx), a transactional client is returned.
func (m TransactionMutation) Client() *Client {
	client := &Client{config: m.config}
	client.init()
	return client
}

// Tx returns an `ent.Tx` for mutations that were executed in transactions;
// it returns an error otherwise.
func (m TransactionMutation) Tx() (*Tx, error) {
	if _, ok := m.driver.(*txDriver); !ok {
		return nil, errors.New("database: mutation is not running in a transaction")
	}
	tx := &Tx{config: m.config}
	tx.init()
	return tx, nil
}

// ID returns the ID value in the mutation. Note that the ID is only available
// if it was provided to the builder or after it was returned from the database.
func (m *TransactionMutation) ID() (id int, exists bool) {
	if m.id == nil {
		return
	}
	return *m.id, true
}

// IDs queries the database and returns the entity ids that match the mutation's predicate.
// That means, if the mutation is applied within a transaction with an isolation level such
// as sql.LevelSerializable, the returned ids match the ids of the rows that will be updated
// or updated by the mutation.
func (m *TransactionMutation) IDs(ctx context.Context) ([]int, error) {
	switch {
	case m.op.Is(OpUpdateOne | OpDeleteOne):
		id, exists := m.ID()
		if exists {
			return []int{id}, nil
		}
		fallthrough
	case m.op.Is(OpUpdate | OpDelete):
		return m.Client().Transaction.Query().Where(m.predicates...).IDs(ctx)
	default:
		return nil, fmt.Errorf("IDs is not allowed on %s operations", m.op)
	}
}

// SetHash sets the "hash" field.
func (m *TransactionMutation) SetHash(s string) {
	m.hash = &s
}

// Hash returns the value of the "hash" field in the mutation.
func (m *TransactionMutation) Hash() (r string, exists bool) {
	v := m.hash
	if v == nil {
		return
	}
	return *v, true
}

// OldHash returns the old "hash" field's value of the Transaction entity.
// If the Transaction object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransactionMutation) OldHash(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldHash is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldHash requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldHash: %w", err)
	}
	return oldValue.Hash, nil
}

// ResetHash resets all changes to the "hash" field.
func (m *TransactionMutation) ResetHash() {
	m.hash = nil
}

// Where appends a list predicates to the TransactionMutation builder.
func (m *TransactionMutation) Where(ps ...predicate.Transaction) {
	m.predicates = append(m.predicates, ps...)
}

// Op returns the operation name.
func (m *TransactionMutation) Op() Op {
	return m.op
}

// Type returns the node type of this mutation (Transaction).
func (m *TransactionMutation) Type() string {
	return m.typ
}

// Fields returns all fields that were changed during this mutation. Note that in
// order to get all numeric fields that were incremented/decremented, call
// AddedFields().
func (m *TransactionMutation) Fields() []string {
	fields := make([]string, 0, 1)
	if m.hash != nil {
		fields = append(fields, transaction.FieldHash)
	}
	return fields
}

// Field returns the value of a field with the given name. The second boolean
// return value indicates that this field was not set, or was not defined in the
// schema.
func (m *TransactionMutation) Field(name string) (ent.Value, bool) {
	switch name {
	case transaction.FieldHash:
		return m.Hash()
	}
	return nil, false
}

// OldField returns the old value of the field from the database. An error is
// returned if the mutation operation is not UpdateOne, or the query to the
// database failed.
func (m *TransactionMutation) OldField(ctx context.Context, name string) (ent.Value, error) {
	switch name {
	case transaction.FieldHash:
		return m.OldHash(ctx)
	}
	return nil, fmt.Errorf("unknown Transaction field %s", name)
}

// SetField sets the value of a field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *TransactionMutation) SetField(name string, value ent.Value) error {
	switch name {
	case transaction.FieldHash:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetHash(v)
		return nil
	}
	return fmt.Errorf("unknown Transaction field %s", name)
}

// AddedFields returns all numeric fields that were incremented/decremented during
// this mutation.
func (m *TransactionMutation) AddedFields() []string {
	return nil
}

// AddedField returns the numeric value that was incremented/decremented on a field
// with the given name. The second boolean return value indicates that this field
// was not set, or was not defined in the schema.
func (m *TransactionMutation) AddedField(name string) (ent.Value, bool) {
	return nil, false
}

// AddField adds the value to the field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *TransactionMutation) AddField(name string, value ent.Value) error {
	switch name {
	}
	return fmt.Errorf("unknown Transaction numeric field %s", name)
}

// ClearedFields returns all nullable fields that were cleared during this
// mutation.
func (m *TransactionMutation) ClearedFields() []string {
	return nil
}

// FieldCleared returns a boolean indicating if a field with the given name was
// cleared in this mutation.
func (m *TransactionMutation) FieldCleared(name string) bool {
	_, ok := m.clearedFields[name]
	return ok
}

// ClearField clears the value of the field with the given name. It returns an
// error if the field is not defined in the schema.
func (m *TransactionMutation) ClearField(name string) error {
	return fmt.Errorf("unknown Transaction nullable field %s", name)
}

// ResetField resets all changes in the mutation for the field with the given name.
// It returns an error if the field is not defined in the schema.
func (m *TransactionMutation) ResetField(name string) error {
	switch name {
	case transaction.FieldHash:
		m.ResetHash()
		return nil
	}
	return fmt.Errorf("unknown Transaction field %s", name)
}

// AddedEdges returns all edge names that were set/added in this mutation.
func (m *TransactionMutation) AddedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// AddedIDs returns all IDs (to other nodes) that were added for the given edge
// name in this mutation.
func (m *TransactionMutation) AddedIDs(name string) []ent.Value {
	return nil
}

// RemovedEdges returns all edge names that were removed in this mutation.
func (m *TransactionMutation) RemovedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// RemovedIDs returns all IDs (to other nodes) that were removed for the edge with
// the given name in this mutation.
func (m *TransactionMutation) RemovedIDs(name string) []ent.Value {
	return nil
}

// ClearedEdges returns all edge names that were cleared in this mutation.
func (m *TransactionMutation) ClearedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// EdgeCleared returns a boolean which indicates if the edge with the given name
// was cleared in this mutation.
func (m *TransactionMutation) EdgeCleared(name string) bool {
	return false
}

// ClearEdge clears the value of the edge with the given name. It returns an error
// if that edge is not defined in the schema.
func (m *TransactionMutation) ClearEdge(name string) error {
	return fmt.Errorf("unknown Transaction unique edge %s", name)
}

// ResetEdge resets all changes to the edge with the given name in this mutation.
// It returns an error if the edge is not defined in the schema.
func (m *TransactionMutation) ResetEdge(name string) error {
	return fmt.Errorf("unknown Transaction edge %s", name)
}

// TransferMutation represents an operation that mutates the Transfer nodes in the graph.
type TransferMutation struct {
	config
	op                       Op
	typ                      string
	id                       *int
	created_at               *time.Time
	updated_at               *time.Time
	transaction_hash         *string
	transaction_log_index    *int
	addtransaction_log_index *int
	address_from             *string
	address_to               *string
	token_address            *string
	token_id                 *string
	clearedFields            map[string]struct{}
	done                     bool
	oldValue                 func(context.Context) (*Transfer, error)
	predicates               []predicate.Transfer
}

var _ ent.Mutation = (*TransferMutation)(nil)

// transferOption allows management of the mutation configuration using functional options.
type transferOption func(*TransferMutation)

// newTransferMutation creates new mutation for the Transfer entity.
func newTransferMutation(c config, op Op, opts ...transferOption) *TransferMutation {
	m := &TransferMutation{
		config:        c,
		op:            op,
		typ:           TypeTransfer,
		clearedFields: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// withTransferID sets the ID field of the mutation.
func withTransferID(id int) transferOption {
	return func(m *TransferMutation) {
		var (
			err   error
			once  sync.Once
			value *Transfer
		)
		m.oldValue = func(ctx context.Context) (*Transfer, error) {
			once.Do(func() {
				if m.done {
					err = errors.New("querying old values post mutation is not allowed")
				} else {
					value, err = m.Client().Transfer.Get(ctx, id)
				}
			})
			return value, err
		}
		m.id = &id
	}
}

// withTransfer sets the old Transfer of the mutation.
func withTransfer(node *Transfer) transferOption {
	return func(m *TransferMutation) {
		m.oldValue = func(context.Context) (*Transfer, error) {
			return node, nil
		}
		m.id = &node.ID
	}
}

// Client returns a new `ent.Client` from the mutation. If the mutation was
// executed in a transaction (ent.Tx), a transactional client is returned.
func (m TransferMutation) Client() *Client {
	client := &Client{config: m.config}
	client.init()
	return client
}

// Tx returns an `ent.Tx` for mutations that were executed in transactions;
// it returns an error otherwise.
func (m TransferMutation) Tx() (*Tx, error) {
	if _, ok := m.driver.(*txDriver); !ok {
		return nil, errors.New("database: mutation is not running in a transaction")
	}
	tx := &Tx{config: m.config}
	tx.init()
	return tx, nil
}

// ID returns the ID value in the mutation. Note that the ID is only available
// if it was provided to the builder or after it was returned from the database.
func (m *TransferMutation) ID() (id int, exists bool) {
	if m.id == nil {
		return
	}
	return *m.id, true
}

// IDs queries the database and returns the entity ids that match the mutation's predicate.
// That means, if the mutation is applied within a transaction with an isolation level such
// as sql.LevelSerializable, the returned ids match the ids of the rows that will be updated
// or updated by the mutation.
func (m *TransferMutation) IDs(ctx context.Context) ([]int, error) {
	switch {
	case m.op.Is(OpUpdateOne | OpDeleteOne):
		id, exists := m.ID()
		if exists {
			return []int{id}, nil
		}
		fallthrough
	case m.op.Is(OpUpdate | OpDelete):
		return m.Client().Transfer.Query().Where(m.predicates...).IDs(ctx)
	default:
		return nil, fmt.Errorf("IDs is not allowed on %s operations", m.op)
	}
}

// SetCreatedAt sets the "created_at" field.
func (m *TransferMutation) SetCreatedAt(t time.Time) {
	m.created_at = &t
}

// CreatedAt returns the value of the "created_at" field in the mutation.
func (m *TransferMutation) CreatedAt() (r time.Time, exists bool) {
	v := m.created_at
	if v == nil {
		return
	}
	return *v, true
}

// OldCreatedAt returns the old "created_at" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldCreatedAt(ctx context.Context) (v time.Time, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldCreatedAt is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldCreatedAt requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldCreatedAt: %w", err)
	}
	return oldValue.CreatedAt, nil
}

// ResetCreatedAt resets all changes to the "created_at" field.
func (m *TransferMutation) ResetCreatedAt() {
	m.created_at = nil
}

// SetUpdatedAt sets the "updated_at" field.
func (m *TransferMutation) SetUpdatedAt(t time.Time) {
	m.updated_at = &t
}

// UpdatedAt returns the value of the "updated_at" field in the mutation.
func (m *TransferMutation) UpdatedAt() (r time.Time, exists bool) {
	v := m.updated_at
	if v == nil {
		return
	}
	return *v, true
}

// OldUpdatedAt returns the old "updated_at" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldUpdatedAt(ctx context.Context) (v time.Time, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldUpdatedAt is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldUpdatedAt requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldUpdatedAt: %w", err)
	}
	return oldValue.UpdatedAt, nil
}

// ResetUpdatedAt resets all changes to the "updated_at" field.
func (m *TransferMutation) ResetUpdatedAt() {
	m.updated_at = nil
}

// SetTransactionHash sets the "transaction_hash" field.
func (m *TransferMutation) SetTransactionHash(s string) {
	m.transaction_hash = &s
}

// TransactionHash returns the value of the "transaction_hash" field in the mutation.
func (m *TransferMutation) TransactionHash() (r string, exists bool) {
	v := m.transaction_hash
	if v == nil {
		return
	}
	return *v, true
}

// OldTransactionHash returns the old "transaction_hash" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldTransactionHash(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldTransactionHash is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldTransactionHash requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldTransactionHash: %w", err)
	}
	return oldValue.TransactionHash, nil
}

// ResetTransactionHash resets all changes to the "transaction_hash" field.
func (m *TransferMutation) ResetTransactionHash() {
	m.transaction_hash = nil
}

// SetTransactionLogIndex sets the "transaction_log_index" field.
func (m *TransferMutation) SetTransactionLogIndex(i int) {
	m.transaction_log_index = &i
	m.addtransaction_log_index = nil
}

// TransactionLogIndex returns the value of the "transaction_log_index" field in the mutation.
func (m *TransferMutation) TransactionLogIndex() (r int, exists bool) {
	v := m.transaction_log_index
	if v == nil {
		return
	}
	return *v, true
}

// OldTransactionLogIndex returns the old "transaction_log_index" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldTransactionLogIndex(ctx context.Context) (v int, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldTransactionLogIndex is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldTransactionLogIndex requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldTransactionLogIndex: %w", err)
	}
	return oldValue.TransactionLogIndex, nil
}

// AddTransactionLogIndex adds i to the "transaction_log_index" field.
func (m *TransferMutation) AddTransactionLogIndex(i int) {
	if m.addtransaction_log_index != nil {
		*m.addtransaction_log_index += i
	} else {
		m.addtransaction_log_index = &i
	}
}

// AddedTransactionLogIndex returns the value that was added to the "transaction_log_index" field in this mutation.
func (m *TransferMutation) AddedTransactionLogIndex() (r int, exists bool) {
	v := m.addtransaction_log_index
	if v == nil {
		return
	}
	return *v, true
}

// ResetTransactionLogIndex resets all changes to the "transaction_log_index" field.
func (m *TransferMutation) ResetTransactionLogIndex() {
	m.transaction_log_index = nil
	m.addtransaction_log_index = nil
}

// SetAddressFrom sets the "address_from" field.
func (m *TransferMutation) SetAddressFrom(s string) {
	m.address_from = &s
}

// AddressFrom returns the value of the "address_from" field in the mutation.
func (m *TransferMutation) AddressFrom() (r string, exists bool) {
	v := m.address_from
	if v == nil {
		return
	}
	return *v, true
}

// OldAddressFrom returns the old "address_from" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldAddressFrom(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldAddressFrom is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldAddressFrom requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldAddressFrom: %w", err)
	}
	return oldValue.AddressFrom, nil
}

// ResetAddressFrom resets all changes to the "address_from" field.
func (m *TransferMutation) ResetAddressFrom() {
	m.address_from = nil
}

// SetAddressTo sets the "address_to" field.
func (m *TransferMutation) SetAddressTo(s string) {
	m.address_to = &s
}

// AddressTo returns the value of the "address_to" field in the mutation.
func (m *TransferMutation) AddressTo() (r string, exists bool) {
	v := m.address_to
	if v == nil {
		return
	}
	return *v, true
}

// OldAddressTo returns the old "address_to" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldAddressTo(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldAddressTo is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldAddressTo requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldAddressTo: %w", err)
	}
	return oldValue.AddressTo, nil
}

// ResetAddressTo resets all changes to the "address_to" field.
func (m *TransferMutation) ResetAddressTo() {
	m.address_to = nil
}

// SetTokenAddress sets the "token_address" field.
func (m *TransferMutation) SetTokenAddress(s string) {
	m.token_address = &s
}

// TokenAddress returns the value of the "token_address" field in the mutation.
func (m *TransferMutation) TokenAddress() (r string, exists bool) {
	v := m.token_address
	if v == nil {
		return
	}
	return *v, true
}

// OldTokenAddress returns the old "token_address" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldTokenAddress(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldTokenAddress is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldTokenAddress requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldTokenAddress: %w", err)
	}
	return oldValue.TokenAddress, nil
}

// ResetTokenAddress resets all changes to the "token_address" field.
func (m *TransferMutation) ResetTokenAddress() {
	m.token_address = nil
}

// SetTokenID sets the "token_id" field.
func (m *TransferMutation) SetTokenID(s string) {
	m.token_id = &s
}

// TokenID returns the value of the "token_id" field in the mutation.
func (m *TransferMutation) TokenID() (r string, exists bool) {
	v := m.token_id
	if v == nil {
		return
	}
	return *v, true
}

// OldTokenID returns the old "token_id" field's value of the Transfer entity.
// If the Transfer object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *TransferMutation) OldTokenID(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldTokenID is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldTokenID requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldTokenID: %w", err)
	}
	return oldValue.TokenID, nil
}

// ResetTokenID resets all changes to the "token_id" field.
func (m *TransferMutation) ResetTokenID() {
	m.token_id = nil
}

// Where appends a list predicates to the TransferMutation builder.
func (m *TransferMutation) Where(ps ...predicate.Transfer) {
	m.predicates = append(m.predicates, ps...)
}

// Op returns the operation name.
func (m *TransferMutation) Op() Op {
	return m.op
}

// Type returns the node type of this mutation (Transfer).
func (m *TransferMutation) Type() string {
	return m.typ
}

// Fields returns all fields that were changed during this mutation. Note that in
// order to get all numeric fields that were incremented/decremented, call
// AddedFields().
func (m *TransferMutation) Fields() []string {
	fields := make([]string, 0, 8)
	if m.created_at != nil {
		fields = append(fields, transfer.FieldCreatedAt)
	}
	if m.updated_at != nil {
		fields = append(fields, transfer.FieldUpdatedAt)
	}
	if m.transaction_hash != nil {
		fields = append(fields, transfer.FieldTransactionHash)
	}
	if m.transaction_log_index != nil {
		fields = append(fields, transfer.FieldTransactionLogIndex)
	}
	if m.address_from != nil {
		fields = append(fields, transfer.FieldAddressFrom)
	}
	if m.address_to != nil {
		fields = append(fields, transfer.FieldAddressTo)
	}
	if m.token_address != nil {
		fields = append(fields, transfer.FieldTokenAddress)
	}
	if m.token_id != nil {
		fields = append(fields, transfer.FieldTokenID)
	}
	return fields
}

// Field returns the value of a field with the given name. The second boolean
// return value indicates that this field was not set, or was not defined in the
// schema.
func (m *TransferMutation) Field(name string) (ent.Value, bool) {
	switch name {
	case transfer.FieldCreatedAt:
		return m.CreatedAt()
	case transfer.FieldUpdatedAt:
		return m.UpdatedAt()
	case transfer.FieldTransactionHash:
		return m.TransactionHash()
	case transfer.FieldTransactionLogIndex:
		return m.TransactionLogIndex()
	case transfer.FieldAddressFrom:
		return m.AddressFrom()
	case transfer.FieldAddressTo:
		return m.AddressTo()
	case transfer.FieldTokenAddress:
		return m.TokenAddress()
	case transfer.FieldTokenID:
		return m.TokenID()
	}
	return nil, false
}

// OldField returns the old value of the field from the database. An error is
// returned if the mutation operation is not UpdateOne, or the query to the
// database failed.
func (m *TransferMutation) OldField(ctx context.Context, name string) (ent.Value, error) {
	switch name {
	case transfer.FieldCreatedAt:
		return m.OldCreatedAt(ctx)
	case transfer.FieldUpdatedAt:
		return m.OldUpdatedAt(ctx)
	case transfer.FieldTransactionHash:
		return m.OldTransactionHash(ctx)
	case transfer.FieldTransactionLogIndex:
		return m.OldTransactionLogIndex(ctx)
	case transfer.FieldAddressFrom:
		return m.OldAddressFrom(ctx)
	case transfer.FieldAddressTo:
		return m.OldAddressTo(ctx)
	case transfer.FieldTokenAddress:
		return m.OldTokenAddress(ctx)
	case transfer.FieldTokenID:
		return m.OldTokenID(ctx)
	}
	return nil, fmt.Errorf("unknown Transfer field %s", name)
}

// SetField sets the value of a field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *TransferMutation) SetField(name string, value ent.Value) error {
	switch name {
	case transfer.FieldCreatedAt:
		v, ok := value.(time.Time)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetCreatedAt(v)
		return nil
	case transfer.FieldUpdatedAt:
		v, ok := value.(time.Time)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetUpdatedAt(v)
		return nil
	case transfer.FieldTransactionHash:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetTransactionHash(v)
		return nil
	case transfer.FieldTransactionLogIndex:
		v, ok := value.(int)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetTransactionLogIndex(v)
		return nil
	case transfer.FieldAddressFrom:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetAddressFrom(v)
		return nil
	case transfer.FieldAddressTo:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetAddressTo(v)
		return nil
	case transfer.FieldTokenAddress:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetTokenAddress(v)
		return nil
	case transfer.FieldTokenID:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetTokenID(v)
		return nil
	}
	return fmt.Errorf("unknown Transfer field %s", name)
}

// AddedFields returns all numeric fields that were incremented/decremented during
// this mutation.
func (m *TransferMutation) AddedFields() []string {
	var fields []string
	if m.addtransaction_log_index != nil {
		fields = append(fields, transfer.FieldTransactionLogIndex)
	}
	return fields
}

// AddedField returns the numeric value that was incremented/decremented on a field
// with the given name. The second boolean return value indicates that this field
// was not set, or was not defined in the schema.
func (m *TransferMutation) AddedField(name string) (ent.Value, bool) {
	switch name {
	case transfer.FieldTransactionLogIndex:
		return m.AddedTransactionLogIndex()
	}
	return nil, false
}

// AddField adds the value to the field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *TransferMutation) AddField(name string, value ent.Value) error {
	switch name {
	case transfer.FieldTransactionLogIndex:
		v, ok := value.(int)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.AddTransactionLogIndex(v)
		return nil
	}
	return fmt.Errorf("unknown Transfer numeric field %s", name)
}

// ClearedFields returns all nullable fields that were cleared during this
// mutation.
func (m *TransferMutation) ClearedFields() []string {
	return nil
}

// FieldCleared returns a boolean indicating if a field with the given name was
// cleared in this mutation.
func (m *TransferMutation) FieldCleared(name string) bool {
	_, ok := m.clearedFields[name]
	return ok
}

// ClearField clears the value of the field with the given name. It returns an
// error if the field is not defined in the schema.
func (m *TransferMutation) ClearField(name string) error {
	return fmt.Errorf("unknown Transfer nullable field %s", name)
}

// ResetField resets all changes in the mutation for the field with the given name.
// It returns an error if the field is not defined in the schema.
func (m *TransferMutation) ResetField(name string) error {
	switch name {
	case transfer.FieldCreatedAt:
		m.ResetCreatedAt()
		return nil
	case transfer.FieldUpdatedAt:
		m.ResetUpdatedAt()
		return nil
	case transfer.FieldTransactionHash:
		m.ResetTransactionHash()
		return nil
	case transfer.FieldTransactionLogIndex:
		m.ResetTransactionLogIndex()
		return nil
	case transfer.FieldAddressFrom:
		m.ResetAddressFrom()
		return nil
	case transfer.FieldAddressTo:
		m.ResetAddressTo()
		return nil
	case transfer.FieldTokenAddress:
		m.ResetTokenAddress()
		return nil
	case transfer.FieldTokenID:
		m.ResetTokenID()
		return nil
	}
	return fmt.Errorf("unknown Transfer field %s", name)
}

// AddedEdges returns all edge names that were set/added in this mutation.
func (m *TransferMutation) AddedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// AddedIDs returns all IDs (to other nodes) that were added for the given edge
// name in this mutation.
func (m *TransferMutation) AddedIDs(name string) []ent.Value {
	return nil
}

// RemovedEdges returns all edge names that were removed in this mutation.
func (m *TransferMutation) RemovedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// RemovedIDs returns all IDs (to other nodes) that were removed for the edge with
// the given name in this mutation.
func (m *TransferMutation) RemovedIDs(name string) []ent.Value {
	return nil
}

// ClearedEdges returns all edge names that were cleared in this mutation.
func (m *TransferMutation) ClearedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// EdgeCleared returns a boolean which indicates if the edge with the given name
// was cleared in this mutation.
func (m *TransferMutation) EdgeCleared(name string) bool {
	return false
}

// ClearEdge clears the value of the edge with the given name. It returns an error
// if that edge is not defined in the schema.
func (m *TransferMutation) ClearEdge(name string) error {
	return fmt.Errorf("unknown Transfer unique edge %s", name)
}

// ResetEdge resets all changes to the edge with the given name in this mutation.
// It returns an error if the edge is not defined in the schema.
func (m *TransferMutation) ResetEdge(name string) error {
	return fmt.Errorf("unknown Transfer edge %s", name)
}
